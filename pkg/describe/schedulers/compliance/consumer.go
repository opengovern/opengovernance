package compliance

import (
	"context"
	"encoding/json"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/runner"
	kafka2 "github.com/kaytu-io/kaytu-util/pkg/kafka"
	"go.uber.org/zap"
	"strings"
	"time"
)

const JobTimeoutCheckInterval = 5 * time.Minute

func (s *JobScheduler) RunComplianceReportJobResultsConsumer() error {
	ctx := context.Background()
	consumer, err := kafka2.NewTopicConsumer(ctx, strings.Split(s.conf.Kafka.Addresses, ","), runner.ResultQueue, runner.ConsumerGroup)
	if err != nil {
		return err
	}
	msgs := consumer.Consume(ctx, s.logger)
	t := time.NewTicker(JobTimeoutCheckInterval)
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
			err := s.db.UpdateRunnerJob(result.Job.ID, result.Status, result.Error)
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
