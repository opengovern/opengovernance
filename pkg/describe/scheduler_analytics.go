package describe

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/kaytu-io/kaytu-engine/pkg/analytics"

	"github.com/kaytu-io/kaytu-util/pkg/queue"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func (s *Scheduler) RunAnalyticsJobScheduler() {
	s.logger.Info("Scheduling analytics jobs on a timer")

	t := time.NewTicker(JobSchedulingInterval)
	defer t.Stop()

	for ; ; <-t.C {
		lastJob, err := s.db.FetchLastAnalyticsJob()
		if err != nil {
			s.logger.Error("Failed to find the last job to check for AnalyticsJob", zap.Error(err))
			continue
		}
		if lastJob == nil || lastJob.CreatedAt.Add(time.Duration(s.analyticsIntervalHours)*time.Hour).Before(time.Now()) {
			err := s.scheduleAnalyticsJob()
			if err != nil {
				s.logger.Error("failure on scheduleAnalyticsJob", zap.Error(err))
			}
		}
	}
}

func (s *Scheduler) scheduleAnalyticsJob() error {
	lastJob, err := s.db.FetchLastAnalyticsJob()
	if err != nil {
		AnalyticsJobsCount.WithLabelValues("failure").Inc()
		s.logger.Error("Failed to get ongoing AnalyticsJob",
			zap.Error(err),
		)
		return err
	}

	if lastJob != nil && lastJob.Status == analytics.JobInProgress {
		s.logger.Info("There is ongoing AnalyticsJob skipping this schedule")
		return fmt.Errorf("there is ongoing AnalyticsJob skipping this schedule")
	}

	job := newAnalyticsJob()

	err = s.db.AddAnalyticsJob(&job)
	if err != nil {
		AnalyticsJobsCount.WithLabelValues("failure").Inc()
		s.logger.Error("Failed to create AnalyticsJob",
			zap.Uint("jobId", job.ID),
			zap.Error(err),
		)
		return err
	}

	err = enqueueAnalyticsJobs(s.db, s.analyticsJobQueue, job)
	if err != nil {
		AnalyticsJobsCount.WithLabelValues("failure").Inc()
		s.logger.Error("Failed to enqueue AnalyticsJob",
			zap.Uint("jobId", job.ID),
			zap.Error(err),
		)
		job.Status = analytics.JobCompletedWithFailure
		err = s.db.UpdateAnalyticsJobStatus(job)
		if err != nil {
			s.logger.Error("Failed to update AnalyticsJob status",
				zap.Uint("jobId", job.ID),
				zap.Error(err),
			)
		}
		return err
	}

	AnalyticsJobsCount.WithLabelValues("successful").Inc()
	return nil
}

func enqueueAnalyticsJobs(db Database, q queue.Interface, job AnalyticsJob) error {
	if err := q.Publish(analytics.Job{
		JobID: job.ID,
	}); err != nil {
		return err
	}

	return nil
}

func newAnalyticsJob() AnalyticsJob {
	return AnalyticsJob{
		Model:          gorm.Model{},
		Status:         analytics.JobCreated,
		FailureMessage: "",
	}
}

func (s *Scheduler) RunAnalyticsJobResultsConsumer() error {
	s.logger.Info("Consuming messages from the analyticsJobResultQueue queue")

	msgs, err := s.analyticsJobResultQueue.Consume()
	if err != nil {
		return err
	}

	t := time.NewTicker(JobTimeoutCheckInterval)
	defer t.Stop()

	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("tasks channel is closed")
			}

			var result analytics.JobResult
			if err := json.Unmarshal(msg.Body, &result); err != nil {
				s.logger.Error("Failed to unmarshal analytics.JobResult results", zap.Error(err))
				err = msg.Nack(false, false)
				if err != nil {
					s.logger.Error("Failed nacking message", zap.Error(err))
				}
				continue
			}

			s.logger.Info("Processing analytics.JobResult for Job",
				zap.Uint("jobId", result.JobID),
				zap.String("status", string(result.Status)),
			)
			err := s.db.UpdateAnalyticsJob(result.JobID, result.Status, result.Error)
			if err != nil {
				s.logger.Error("Failed to update the status of AnalyticsJob",
					zap.Uint("jobId", result.JobID),
					zap.Error(err))
				err = msg.Nack(false, true)
				if err != nil {
					s.logger.Error("Failed nacking message", zap.Error(err))
				}
				continue
			}

			if err := msg.Ack(false); err != nil {
				s.logger.Error("Failed acking message", zap.Error(err))
			}
		case <-t.C:
			err := s.db.UpdateAnalyticsJobsTimedOut(s.analyticsIntervalHours)
			if err != nil {
				s.logger.Error("Failed to update timed out AnalyticsJob", zap.Error(err))
			}
		}
	}
}
