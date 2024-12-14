package scheduler

import (
	"encoding/json"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/opengovern/opencomply/services/tasks/db/models"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

type TaskResponse struct {
	RunID          uint                 `json:"run_id"`
	Status         models.TaskRunStatus `json:"status"`
	FailureMessage string               `json:"failure_message"`
	Result         []byte               `json:"result"`
}

func (s *TaskScheduler) RunTaskResponseConsumer(ctx context.Context) error {
	if _, err := s.jq.Consume(ctx, s.NatsConfig.ResultConsumer, s.NatsConfig.Stream, []string{s.NatsConfig.ResultTopic},
		s.NatsConfig.ResultConsumer, func(msg jetstream.Msg) {
			if err := msg.Ack(); err != nil {
				s.logger.Error("Failed committing message", zap.Error(err))
			}

			var response TaskResponse
			if err := json.Unmarshal(msg.Data(), &response); err != nil {
				s.logger.Error("Failed to unmarshal ComplianceReportJob results", zap.Error(err))
				return
			}

			taskRunUpdate := models.TaskRun{
				Status:         response.Status,
				FailureMessage: response.FailureMessage,
			}

			if response.Status == models.TaskRunStatusFinished {
				s.logger.Info("Processing RunTaskResponse for Task",
					zap.String("Task", s.TaskID),
					zap.Uint("RunID", response.RunID),
					zap.String("Result", string(response.Result)),
				)

				taskRunUpdate.Result = string(response.Result)
			}
			err := s.db.UpdateTaskRun(response.RunID, taskRunUpdate.Status, taskRunUpdate.Result)
			if err != nil {
				s.logger.Error("Failed to update the status of RunTaskResponse",
					zap.String("Task", s.TaskID),
					zap.Uint("RunID", response.RunID),
					zap.Error(err))
				return
			}
		}); err != nil {
		return err
	}

	<-ctx.Done()
	return nil
}
