package describe

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/opengovern/og-util/pkg/api"
	"github.com/opengovern/og-util/pkg/httpclient"
	integrationapi "github.com/opengovern/opengovernance/services/integration/api/models"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/opengovern/og-util/pkg/ticker"
	"github.com/opengovern/opengovernance/pkg/analytics"
	analyticsApi "github.com/opengovern/opengovernance/pkg/analytics/api"
	"github.com/opengovern/opengovernance/pkg/describe/db/model"
	inventoryApi "github.com/opengovern/opengovernance/services/inventory/api"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func (s *Scheduler) RunAnalyticsJobScheduler(ctx context.Context) {
	s.logger.Info("Scheduling analytics jobs on a timer")

	t := ticker.NewTicker(JobSchedulingInterval, time.Second*10)
	defer t.Stop()
	ctx2 := &httpclient.Context{UserRole: api.AdminRole}
	ctx2.Ctx = ctx
	for ; ; <-t.C {
		integrations, err := s.integrationClient.ListIntegrations(ctx2, nil)
		if err != nil {
			s.logger.Error("Failed to list sources", zap.Error(err))
			AnalyticsJobsCount.WithLabelValues("failure").Inc()
			continue
		}
		hasEnabled := false
		for _, integration := range integrations.Integrations {
			if integration.State == integrationapi.IntegrationStateActive {
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
			_, err := s.scheduleAnalyticsJob(model.AnalyticsJobTypeNormal, ctx)
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
			_, err := s.scheduleAnalyticsJob(model.AnalyticsJobTypeResourceCollection, ctx)
			if err != nil {
				s.logger.Error("failure on scheduleAnalyticsJob", zap.Error(err))
			}
		}

	}
}

func (s *Scheduler) scheduleAnalyticsJob(analyticsJobType model.AnalyticsJobType, ctx context.Context) (uint, error) {
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

	if err = s.db.AddAnalyticsJob(&job); err != nil {
		AnalyticsJobsCount.WithLabelValues("failure").Inc()
		s.logger.Error("Failed to create AnalyticsJob",
			zap.Uint("jobId", job.ID),
			zap.Error(err),
		)
		return 0, err
	}

	if err = s.enqueueAnalyticsJobs(job, ctx); err != nil {
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

func (s *Scheduler) enqueueAnalyticsJobs(job model.AnalyticsJob, ctx context.Context) error {
	var resourceCollectionIds []string

	if job.Type == model.AnalyticsJobTypeResourceCollection {
		resourceCollections, err := s.inventoryClient.ListResourceCollections(&httpclient.Context{UserRole: api.AdminRole})
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

	if _, err := s.jq.Produce(ctx, analytics.JobQueueTopic, aJobJson, fmt.Sprintf("job-%d", job.ID)); err != nil {
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

func (s *Scheduler) RunAnalyticsJobResultsConsumer(ctx context.Context) error {
	s.logger.Info("Consuming messages from the analytics Job Result Queue queue")

	consumeCtx, err := s.jq.Consume(ctx, "analytics-scheduler", analytics.StreamName, []string{analytics.JobResultQueueTopic}, "analytics-scheduler", func(msg jetstream.Msg) {
		var result analytics.JobResult

		if err := json.Unmarshal(msg.Data(), &result); err != nil {
			AnalyticsJobResultsCount.WithLabelValues("failure").Inc()
			s.logger.Error("Failed to unmarshal analytics.JobResult results", zap.Error(err), zap.ByteString("value", msg.Data()))

			if err := msg.Ack(); err != nil {
				s.logger.Error("Failed to commit message", zap.Error(err))
			}

			return
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

		if err := s.db.UpdateAnalyticsJob(result.JobID, result.Status, result.Error); err != nil {
			AnalyticsJobResultsCount.WithLabelValues("failure").Inc()
			s.logger.Error("Failed to update the status of AnalyticsJob",
				zap.Uint("jobId", result.JobID),
				zap.Error(err),
			)

			return
		}

		if err := msg.Ack(); err != nil {
			AnalyticsJobResultsCount.WithLabelValues("failure").Inc()
			s.logger.Error("Failed to commit message", zap.Error(err))
		}
	})
	if err != nil {
		s.logger.Error("Failed to create nats consumer", zap.Error(err))
		return err
	}

	tick := ticker.NewTicker(5*time.Minute, time.Minute)
	for {
		select {
		case <-tick.C:
			if err := s.db.UpdateAnalyticsJobsTimedOut(); err != nil {
				s.logger.Error("failed to update analytics job timeout", zap.Error(err))
			}
		case <-ctx.Done():
			consumeCtx.Drain()
			consumeCtx.Stop()
			tick.Stop()
			return nil
		}
	}
}
