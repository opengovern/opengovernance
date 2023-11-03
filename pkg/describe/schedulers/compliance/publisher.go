package compliance

import (
	"encoding/json"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/runner"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	kafka2 "github.com/kaytu-io/kaytu-util/pkg/kafka"
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

func (s *JobScheduler) runPublisher() error {
	s.logger.Info("runPublisher")
	ctx := &httpclient.Context{UserRole: api.InternalRole}

	for i := 0; i < 10; i++ {
		runners, err := s.db.FetchCreatedRunners()
		if err != nil {
			return err
		}

		if len(runners) == 0 {
			return nil
		}

		for _, it := range runners {
			query, err := s.complianceClient.GetQuery(ctx, it.QueryID)
			if err != nil {
				return err
			}

			callers, err := it.GetCallers()
			if err != nil {
				return err
			}

			job := runner.Job{
				ID:        it.ID,
				CreatedAt: it.CreatedAt,
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
				_ = s.db.UpdateRunnerJob(job.ID, runner.ComplianceRunnerFailed, err.Error())
				return err
			}

			msg := kafka2.Msg(fmt.Sprintf("job-%d", job.ID), jobJson, "", runner.JobQueue, kafka.PartitionAny)
			_, err = kafka2.SyncSend(s.logger, s.kafkaProducer, []*kafka.Message{msg}, nil)
			if err != nil {
				_ = s.db.UpdateRunnerJob(job.ID, runner.ComplianceRunnerFailed, err.Error())
				return err
			}

			_ = s.db.UpdateRunnerJob(job.ID, runner.ComplianceRunnerInProgress, "")
		}
	}

	err := s.db.RetryFailedRunners()
	if err != nil {
		return err
	}

	return nil
}
