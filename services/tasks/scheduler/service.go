package scheduler

import (
	"encoding/json"
	"fmt"
	"github.com/jackc/pgtype"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/opengovern/og-util/pkg/jq"
	"github.com/opengovern/opencomply/services/tasks/db"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"time"
)

type MainScheduler struct {
	jq     *jq.JobQueue
	db     db.Database
	logger *zap.Logger

	Tasks []TaskScheduler
}

var RunningTasks = make(map[string]bool)

func NewMainScheduler(logger *zap.Logger, db db.Database, jq *jq.JobQueue) *MainScheduler {
	return &MainScheduler{
		jq:     jq,
		db:     db,
		logger: logger,
	}
}

func (s *MainScheduler) Start(ctx context.Context) error {
	tasks, err := s.db.GetTaskList()
	if err != nil {
		s.logger.Error("failed to get task list", zap.Error(err))
		return err
	}

	for _, task := range tasks {
		if _, ok := RunningTasks[task.ID]; ok {
			continue
		}
		var natsConfig NatsConfig
		if task.NatsConfig.Status != pgtype.Present {
			return fmt.Errorf("JSONB data is not present")
		}
		if err := json.Unmarshal(task.NatsConfig.Bytes, &natsConfig); err != nil {
			return fmt.Errorf("failed to unmarshal JSONB: %w", err)
		}

		err = s.SetupNats(ctx, task.ID, natsConfig)
		if err != nil {
			s.logger.Error("Failed to setup nats streams", zap.Error(err))
			return err
		}

		taskScheduler := NewTaskScheduler(
			func(ctx context.Context) error {
				return s.SetupNats(ctx, task.ID, natsConfig)
			},
			s.logger,
			s.db,
			s.jq,
			task.ID,
			natsConfig,
			task.Interval)
		taskScheduler.Run(ctx)
		RunningTasks[task.ID] = true
	}
	return nil
}

func (s *MainScheduler) SetupNats(ctx context.Context, taskName string, natsConfig NatsConfig) error {
	if err := s.jq.Stream(ctx, natsConfig.Stream, "Query Runner job queues", []string{natsConfig.Topic, natsConfig.ResultTopic}, 100); err != nil {
		s.logger.Error("Failed to stream to task queue", zap.String("task", taskName), zap.Error(err))
		return err
	}
	if err := s.jq.CreateOrUpdateConsumer(ctx, natsConfig.Consumer, natsConfig.Stream,
		[]string{natsConfig.Topic}, jetstream.ConsumerConfig{
			Replicas:          1,
			AckPolicy:         jetstream.AckExplicitPolicy,
			DeliverPolicy:     jetstream.DeliverAllPolicy,
			MaxAckPending:     -1,
			AckWait:           time.Minute * 30,
			InactiveThreshold: time.Hour,
		}); err != nil {
		s.logger.Error("Failed to create consumer for task", zap.String("task", taskName), zap.Error(err))
		return err
	}
	return nil
}