package compliance

import (
	"context"
	"encoding/json"
	"fmt"
	onboardApi "github.com/kaytu-io/kaytu-engine/pkg/onboard/api"

	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/runner"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	"go.uber.org/zap"
)

func (s *JobScheduler) runPublisher() error {
	s.logger.Info("runPublisher")
	ctx := &httpclient.Context{UserRole: api.InternalRole}

	connectionsMap := make(map[string]*onboardApi.Connection)
	connections, err := s.onboardClient.ListSources(ctx, nil)
	if err != nil {
		s.logger.Error("failed to get connections", zap.Error(err))
		return err
	}
	for _, connection := range connections {
		connection := connection
		connectionsMap[connection.ID.String()] = &connection
	}

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
			if query == nil {
				s.logger.Error("query not found", zap.String("queryId", it.QueryID), zap.Uint("runnerId", it.ID))
				continue
			}

			callers, err := it.GetCallers()
			if err != nil {
				s.logger.Error("failed to get callers", zap.Error(err), zap.Uint("runnerId", it.ID))
				continue
			}
			var providerConnectionID *string
			if it.ConnectionID != nil && *it.ConnectionID != "" {
				providerConnectionID = &connectionsMap[*it.ConnectionID].ConnectionID
			}
			job := runner.Job{
				ID:          it.ID,
				ParentJobID: it.ParentJobID,
				CreatedAt:   it.CreatedAt,
				ExecutionPlan: runner.ExecutionPlan{
					Callers:              callers,
					Query:                *query,
					ConnectionID:         it.ConnectionID,
					ProviderConnectionID: providerConnectionID,
				},
			}

			jobJson, err := json.Marshal(job)
			if err != nil {
				_ = s.db.UpdateRunnerJob(job.ID, runner.ComplianceRunnerFailed, job.CreatedAt, nil, err.Error())
				s.logger.Error("failed to marshal job", zap.Error(err), zap.Uint("runnerId", it.ID))
				continue
			}

			if err := s.jq.Produce(context.Background(), runner.JobQueueTopic, jobJson, fmt.Sprintf("job-%d", job.ID)); err != nil {
				_ = s.db.UpdateRunnerJob(job.ID, runner.ComplianceRunnerFailed, job.CreatedAt, nil, err.Error())
				s.logger.Error("failed to send job", zap.Error(err), zap.Uint("runnerId", it.ID))
				continue
			}

			_ = s.db.UpdateRunnerJob(job.ID, runner.ComplianceRunnerInProgress, job.CreatedAt, nil, "")
		}
	}

	err = s.db.RetryFailedRunners()
	if err != nil {
		s.logger.Error("failed to retry failed runners", zap.Error(err))
		return err
	}

	return nil
}
