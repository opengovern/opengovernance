package scheduler

import (
	"encoding/json"
	"fmt"
	"github.com/jackc/pgtype"
	"github.com/opengovern/og-util/pkg/api"
	"github.com/opengovern/og-util/pkg/httpclient"
	"github.com/opengovern/opencomply/services/tasks/db/models"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

type TaskRequest struct {
	RunID  uint              `json:"run_id"`
	Params map[string]string `json:"params"`
}

func (s *TaskScheduler) runPublisher(ctx context.Context) error {
	ctx2 := &httpclient.Context{UserRole: api.AdminRole}
	ctx2.Ctx = ctx

	s.logger.Info("Query Runner publisher started")

	runs, err := s.db.FetchCreatedTaskRunsByTaskID(s.TaskID)
	if err != nil {
		s.logger.Error("failed to get task runs", zap.Error(err))
		return err
	}

	for _, run := range runs {
		params, err := JSONBToMap(run.Params)
		if err != nil {
			_ = s.db.UpdateTaskRun(run.ID, models.TaskRunStatusFailed, "", "failed to get params")
			s.logger.Error("failed to get params", zap.Error(err), zap.Uint("runId", run.ID))
			return err
		}
		req := TaskRequest{
			RunID:  run.ID,
			Params: params,
		}
		reqJson, err := json.Marshal(req)
		if err != nil {
			_ = s.db.UpdateTaskRun(run.ID, models.TaskRunStatusFailed, "", "failed to marshal run")
			s.logger.Error("failed to marshal Task Run", zap.Error(err), zap.Uint("runId", run.ID))
			return err
		}

		s.logger.Info("publishing audit job", zap.Uint("runId", run.ID))
		_, err = s.jq.Produce(ctx, s.NatsConfig.Topic, reqJson, fmt.Sprintf("run-%d", run.ID))
		if err != nil {
			if err.Error() == "nats: no response from stream" {
				err = s.runSetupNatsStreams(ctx)
				if err != nil {
					s.logger.Error("Failed to setup nats streams", zap.Error(err))
					return err
				}
				_, err = s.jq.Produce(ctx, s.NatsConfig.Topic, reqJson, fmt.Sprintf("run-%d", run.ID))
				if err != nil {
					_ = s.db.UpdateTaskRun(run.ID, models.TaskRunStatusFailed, "", err.Error())
					s.logger.Error("failed to send run", zap.Error(err), zap.Uint("runId", run.ID))
					continue
				}
			} else {
				_ = s.db.UpdateTaskRun(run.ID, models.TaskRunStatusFailed, "", err.Error())
				s.logger.Error("failed to send run", zap.Error(err), zap.Uint("runId", run.ID), zap.String("error message", err.Error()))
				continue
			}
		} else {
			_ = s.db.UpdateTaskRun(run.ID, models.TaskRunStatusQueued, "", "")
		}
	}

	return nil
}

func JSONBToMap(jsonb pgtype.JSONB) (map[string]string, error) {
	if jsonb.Status != pgtype.Present {
		return nil, fmt.Errorf("JSONB data is not present")
	}

	var result map[string]string
	if err := json.Unmarshal(jsonb.Bytes, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSONB: %w", err)
	}

	return result, nil
}
