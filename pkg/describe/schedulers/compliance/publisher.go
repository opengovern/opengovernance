package compliance

import (
	"encoding/json"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/runner"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	kafka2 "github.com/kaytu-io/kaytu-util/pkg/kafka"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"go.uber.org/zap"
)

func (s *JobScheduler) runPublisher() error {
	s.logger.Info("runPublisher")
	ctx := &httpclient.Context{UserRole: api.InternalRole}

	for i := 0; i < 10; i++ {
		err := s.db.UpdateTimedOutRunners()
		if err != nil {
			s.logger.Error("failed to update timed out runners", zap.Error(err))
		}
		runners, err := s.db.FetchCreatedRunners()
		if err != nil {
			s.logger.Error("failed to fetch created runners", zap.Error(err))
			continue
		}

		if len(runners) == 0 {
			break
		}

		for _, it := range runners {
			query, err := s.complianceClient.GetQuery(ctx, it.QueryID)
			if err != nil {
				s.logger.Error("failed to get query", zap.Error(err), zap.String("queryId", it.QueryID), zap.Uint("runnerId", it.ID))
				continue
			}

			callers, err := it.GetCallers()
			if err != nil {
				s.logger.Error("failed to get callers", zap.Error(err), zap.Uint("runnerId", it.ID))
				continue
			}

			job := runner.Job{
				ID:          it.ID,
				ParentJobID: it.ParentJobID,
				CreatedAt:   it.CreatedAt,
				ExecutionPlan: runner.ExecutionPlan{
					Callers:              callers,
					QueryID:              it.QueryID,
					QueryEngine:          query.Engine,
					QueryConnector:       source.Type(query.Connector),
					ConnectionID:         it.ConnectionID,
					ResourceCollectionID: it.ResourceCollectionID,
				},
			}

			jobJson, err := json.Marshal(job)
			if err != nil {
				_ = s.db.UpdateRunnerJob(job.ID, runner.ComplianceRunnerFailed, job.CreatedAt, nil, err.Error())
				s.logger.Error("failed to marshal job", zap.Error(err), zap.Uint("runnerId", it.ID))
				continue
			}

			msg := kafka2.Msg(fmt.Sprintf("job-%d", job.ID), jobJson, "", runner.JobQueue, kafka.PartitionAny)
			_, err = kafka2.SyncSend(s.logger, s.kafkaProducer, []*kafka.Message{msg}, nil)
			if err != nil {
				_ = s.db.UpdateRunnerJob(job.ID, runner.ComplianceRunnerFailed, job.CreatedAt, nil, err.Error())
				s.logger.Error("failed to send job", zap.Error(err), zap.Uint("runnerId", it.ID))
				continue
			}

			_ = s.db.UpdateRunnerJob(job.ID, runner.ComplianceRunnerInProgress, job.CreatedAt, nil, "")
		}
	}

	err := s.db.RetryFailedRunners()
	if err != nil {
		s.logger.Error("failed to retry failed runners", zap.Error(err))
		return err
	}

	return nil
}
