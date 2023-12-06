package describe

import (
	"context"
	"encoding/json"
	"fmt"
	confluent_kafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	analyticsApi "github.com/kaytu-io/kaytu-engine/pkg/analytics/api"
	authApi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/db/model"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	inventoryApi "github.com/kaytu-io/kaytu-engine/pkg/inventory/api"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"
	"time"

	"github.com/kaytu-io/kaytu-engine/pkg/analytics"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func (s *Scheduler) RunAnalyticsJobScheduler() {
	s.logger.Info("Scheduling analytics jobs on a timer")

	t := time.NewTicker(JobSchedulingInterval)
	defer t.Stop()

	for ; ; <-t.C {
		connections, err := s.onboardClient.ListSources(&httpclient.Context{UserRole: authApi.InternalRole}, nil)
		if err != nil {
			s.logger.Error("Failed to list sources", zap.Error(err))
			AnalyticsJobsCount.WithLabelValues("failure").Inc()
			continue
		}
		hasEnabled := false
		for _, connection := range connections {
			if connection.IsEnabled() {
				hasEnabled = true
				break
			}
		}
		if !hasEnabled {
			s.logger.Info("No enabled sources found, skipping analytics job scheduling")
			continue
		}

		lastJob, err := s.db.FetchLastAnalyticsJobForJobType(model.AnalyticsJobTypeNormal)
		if err != nil {
			s.logger.Error("Failed to find the last job to check for AnalyticsJob", zap.Error(err))
			AnalyticsJobsCount.WithLabelValues("failure").Inc()
			continue
		}
		if lastJob == nil || lastJob.CreatedAt.Add(s.analyticsIntervalHours).Before(time.Now()) {
			_, err := s.scheduleAnalyticsJob(model.AnalyticsJobTypeNormal)
			if err != nil {
				s.logger.Error("failure on scheduleAnalyticsJob", zap.Error(err))
			}
		}

		lastJob, err = s.db.FetchLastAnalyticsJobForJobType(model.AnalyticsJobTypeResourceCollection)
		if err != nil {
			s.logger.Error("Failed to find the last job to check for AnalyticsJob on resourceCollection", zap.Error(err))
			AnalyticsJobsCount.WithLabelValues("failure").Inc()
			continue
		}
		if lastJob == nil || lastJob.CreatedAt.Add(s.analyticsIntervalHours).Before(time.Now()) {
			_, err := s.scheduleAnalyticsJob(model.AnalyticsJobTypeResourceCollection)
			if err != nil {
				s.logger.Error("failure on scheduleAnalyticsJob", zap.Error(err))
			}
		}

	}
}

func (s *Scheduler) scheduleAnalyticsJob(analyticsJobType model.AnalyticsJobType) (uint, error) {
	lastJob, err := s.db.FetchLastAnalyticsJobForJobType(analyticsJobType)
	if err != nil {
		AnalyticsJobsCount.WithLabelValues("failure").Inc()
		s.logger.Error("Failed to get ongoing AnalyticsJob",
			zap.Error(err),
		)
		return 0, err
	}

	if lastJob != nil && lastJob.Status == analyticsApi.JobInProgress {
		s.logger.Info("There is ongoing AnalyticsJob skipping this schedule")
		return 0, fmt.Errorf("there is ongoing AnalyticsJob skipping this schedule")
	}

	job := newAnalyticsJob(analyticsJobType)

	err = s.db.AddAnalyticsJob(&job)
	if err != nil {
		AnalyticsJobsCount.WithLabelValues("failure").Inc()
		s.logger.Error("Failed to create AnalyticsJob",
			zap.Uint("jobId", job.ID),
			zap.Error(err),
		)
		return 0, err
	}

	err = s.enqueueAnalyticsJobs(job)
	if err != nil {
		AnalyticsJobsCount.WithLabelValues("failure").Inc()
		s.logger.Error("Failed to enqueue AnalyticsJob",
			zap.Uint("jobId", job.ID),
			zap.Error(err),
		)
		job.Status = analyticsApi.JobCompletedWithFailure
		err = s.db.UpdateAnalyticsJobStatus(job)
		if err != nil {
			s.logger.Error("Failed to update AnalyticsJob status",
				zap.Uint("jobId", job.ID),
				zap.Error(err),
			)
		}
		return 0, err
	}

	AnalyticsJobsCount.WithLabelValues("successful").Inc()
	return job.ID, nil
}

func (s *Scheduler) enqueueAnalyticsJobs(job model.AnalyticsJob) error {
	var resourceCollectionIds []string

	if job.Type == model.AnalyticsJobTypeResourceCollection {
		resourceCollections, err := s.inventoryClient.ListResourceCollections(&httpclient.Context{UserRole: authApi.InternalRole})
		if err != nil {
			s.logger.Error("Failed to list resource collections", zap.Error(err))
			return err
		}
		for _, resourceCollection := range resourceCollections {
			if resourceCollection.Status != inventoryApi.ResourceCollectionStatusActive {
				continue
			}
			resourceCollectionIds = append(resourceCollectionIds, resourceCollection.ID)
		}
	}

	aJobJson, err := json.Marshal(analytics.Job{
		JobID:                 job.ID,
		ResourceCollectionIDs: resourceCollectionIds,
	})
	if err != nil {
		s.logger.Error("Failed to marshal analytics.Job", zap.Error(err))
		return err
	}

	if err := kafka.SyncSendWithRetry(s.logger, s.kafkaProducer, []*confluent_kafka.Message{{
		TopicPartition: confluent_kafka.TopicPartition{Topic: utils.GetPointer(analytics.JobQueueTopic), Partition: confluent_kafka.PartitionAny},
		Value:          aJobJson,
	}}, nil, 5); err != nil {
		return err
	}

	return nil
}

func newAnalyticsJob(analyticsJobType model.AnalyticsJobType) model.AnalyticsJob {
	return model.AnalyticsJob{
		Model:          gorm.Model{},
		Type:           analyticsJobType,
		Status:         analyticsApi.JobCreated,
		FailureMessage: "",
	}
}

func (s *Scheduler) RunAnalyticsJobResultsConsumer() error {
	s.logger.Info("Consuming messages from the analyticsJobResultQueue queue")

	consumer, err := kafka.NewTopicConsumer(
		context.Background(),
		s.kafkaServers,
		analytics.JobResultQueueTopic,
		schedulerConsumerGroup,
		false,
	)
	if err != nil {
		s.logger.Error("Failed to create kafka consumer", zap.Error(err))
		return err
	}
	defer consumer.Close()

	msgs := consumer.Consume(context.TODO(), s.logger, 100)
	t := time.NewTicker(JobTimeoutCheckInterval)
	defer t.Stop()

	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("tasks channel is closed")
			}

			var result analytics.JobResult
			if err := json.Unmarshal(msg.Value, &result); err != nil {
				AnalyticsJobResultsCount.WithLabelValues("failure").Inc()
				s.logger.Error("Failed to unmarshal analytics.JobResult results", zap.Error(err), zap.String("value", string(msg.Value)))
				err = consumer.Commit(msg)
				if err != nil {
					s.logger.Error("Failed to commit message", zap.Error(err))
				}
				continue
			}

			s.logger.Info("Processing analytics.JobResult for Job",
				zap.Uint("jobId", result.JobID),
				zap.String("status", string(result.Status)),
			)

			if result.Status == analyticsApi.JobCompleted {
				AnalyticsJobResultsCount.WithLabelValues("successful").Inc()
			} else {
				AnalyticsJobResultsCount.WithLabelValues("failure").Inc()
			}

			err := s.db.UpdateAnalyticsJob(result.JobID, result.Status, result.Error)
			if err != nil {
				AnalyticsJobResultsCount.WithLabelValues("failure").Inc()
				s.logger.Error("Failed to update the status of AnalyticsJob",
					zap.Uint("jobId", result.JobID),
					zap.Error(err))

				if err := consumer.Commit(msg); err != nil {
					AnalyticsJobResultsCount.WithLabelValues("failure").Inc()
					s.logger.Error("Failed to commit message", zap.Error(err))
				}

				// requeue the message
				if err2 := kafka.SyncSendWithRetry(s.logger, s.kafkaProducer, []*confluent_kafka.Message{{
					TopicPartition: confluent_kafka.TopicPartition{Topic: utils.GetPointer(analytics.JobResultQueueTopic), Partition: confluent_kafka.PartitionAny},
					Value:          msg.Value,
				}}, nil, 5); err2 != nil {
					s.logger.Error("Failed to requeue the message", zap.Error(err2))
					return err
				}
				continue
			}

			if err := consumer.Commit(msg); err != nil {
				AnalyticsJobResultsCount.WithLabelValues("failure").Inc()
				s.logger.Error("Failed to commit message", zap.Error(err))
			}
		case <-t.C:
			err := s.db.UpdateAnalyticsJobsTimedOut(s.analyticsIntervalHours)
			if err != nil {
				AnalyticsJobResultsCount.WithLabelValues("failure").Inc()
				s.logger.Error("Failed to update timed out AnalyticsJob", zap.Error(err))
			}
		}
	}
}
