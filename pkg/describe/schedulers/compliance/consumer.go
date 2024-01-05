package compliance

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/kaytu-io/kaytu-engine/pkg/compliance/runner"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/summarizer"
	kafka2 "github.com/kaytu-io/kaytu-util/pkg/kafka"
	"github.com/kaytu-io/kaytu-util/pkg/ticker"
	"go.uber.org/zap"
)

const JobTimeoutCheckInterval = 5 * time.Minute

func (s *JobScheduler) RunComplianceReportJobResultsConsumer() error {
	ctx := context.Background()
	consumer, err := kafka2.NewTopicConsumer(ctx, strings.Split(s.conf.Kafka.Addresses, ","), runner.ResultQueueTopic, runner.ConsumerGroup, false)
	if err != nil {
		return err
	}
	msgs := consumer.Consume(ctx, s.logger, 100)
	t := ticker.NewTicker(JobTimeoutCheckInterval, time.Second*10)
	defer t.Stop()

	for {
		select {
		case msg := <-msgs:
			var result runner.JobResult
			if err := json.Unmarshal(msg.Value, &result); err != nil {
				s.logger.Error("Failed to unmarshal ComplianceReportJob results", zap.Error(err))

				err := consumer.Commit(msg)
				if err != nil {
					s.logger.Error("Failed committing message", zap.Error(err))
				}
				continue
			}

			s.logger.Info("Processing ReportJobResult for Job",
				zap.Uint("jobId", result.Job.ID),
				zap.String("status", string(result.Status)),
			)
			err := s.db.UpdateRunnerJob(result.Job.ID, result.Status, result.StartedAt, result.TotalFindingCount, result.Error)
			if err != nil {
				s.logger.Error("Failed to update the status of ComplianceReportJob",
					zap.Uint("jobId", result.Job.ID),
					zap.Error(err))

				err := consumer.Commit(msg)
				if err != nil {
					s.logger.Error("Failed committing message", zap.Error(err))
				}
				continue
			}

			err = consumer.Commit(msg)
			if err != nil {
				s.logger.Error("Failed committing message", zap.Error(err))
			}
		case <-t.C:
			//err := s.db.UpdateRunnerJobsTimedOut()
			//if err != nil {
			//	s.logger.Error("Failed to update timed out ComplianceReportJob", zap.Error(err))
			//}
		}
	}
}

func (s *JobScheduler) RunComplianceSummarizerResultsConsumer() error {
	ctx := context.Background()
	consumer, err := kafka2.NewTopicConsumer(ctx, strings.Split(s.conf.Kafka.Addresses, ","), summarizer.ResultQueueTopic, summarizer.ConsumerGroup, false)
	if err != nil {
		return err
	}
	msgs := consumer.Consume(ctx, s.logger, 100)
	t := ticker.NewTicker(JobTimeoutCheckInterval, time.Second*10)
	defer t.Stop()

	for {
		select {
		case msg := <-msgs:
			var result summarizer.JobResult
			if err := json.Unmarshal(msg.Value, &result); err != nil {
				s.logger.Error("Failed to unmarshal ComplianceSummarizer results", zap.Error(err))

				err := consumer.Commit(msg)
				if err != nil {
					s.logger.Error("Failed committing message", zap.Error(err))
				}
				continue
			}

			s.logger.Info("Processing SummarizerResult for Job",
				zap.Uint("jobId", result.Job.ID),
				zap.String("status", string(result.Status)),
			)
			err := s.db.UpdateSummarizerJob(result.Job.ID, result.Status, result.StartedAt, result.Error)
			if err != nil {
				s.logger.Error("Failed to update the status of Summarizer",
					zap.Uint("jobId", result.Job.ID),
					zap.Error(err))

				err := consumer.Commit(msg)
				if err != nil {
					s.logger.Error("Failed committing message", zap.Error(err))
				}
				continue
			}

			err = consumer.Commit(msg)
			if err != nil {
				s.logger.Error("Failed committing message", zap.Error(err))
			}
		case <-t.C:
			//err := s.db.UpdateRunnerJobsTimedOut()
			//if err != nil {
			//	s.logger.Error("Failed to update timed out ComplianceReportJob", zap.Error(err))
			//}
		}
	}
}
