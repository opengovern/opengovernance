package scheduler

import (
	"github.com/opengovern/og-util/pkg/jq"
	"github.com/opengovern/og-util/pkg/ticker"
	"github.com/opengovern/opencomply/pkg/utils"
	"github.com/opengovern/opencomply/services/tasks/db"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"time"
)

type NatsConfig struct {
	Stream         string
	Topic          string
	ResultTopic    string
	Consumer       string
	ResultConsumer string
}

type TaskScheduler struct {
	runSetupNatsStreams func(context.Context) error
	jq                  *jq.JobQueue
	db                  db.Database
	logger              *zap.Logger

	TaskName   string
	NatsConfig NatsConfig
	Interval   uint64
	Timeout    uint
}

func NewTaskScheduler(
	runSetupNatsStreams func(context.Context) error,
	logger *zap.Logger,
	db db.Database,
	jq *jq.JobQueue,

	taskName string, natsConfig NatsConfig, interval uint64) *TaskScheduler {
	return &TaskScheduler{
		runSetupNatsStreams: runSetupNatsStreams,
		logger:              logger,
		db:                  db,
		jq:                  jq,

		TaskName:   taskName,
		NatsConfig: natsConfig,
		Interval:   interval,
	}
}

func (s *TaskScheduler) Run(ctx context.Context) {
	s.logger.Info("Run task scheduler started", zap.String("task", s.TaskName),
		zap.Any("nats config", s.NatsConfig), zap.Uint64("interval", s.Interval))
	utils.EnsureRunGoroutine(func() {
		s.RunPublisher(ctx)
	})
	utils.EnsureRunGoroutine(func() {
		s.logger.Fatal("RunTaskResponseConsumer exited", zap.Error(s.RunTaskResponseConsumer(ctx)))
	})
}

func (s *TaskScheduler) RunPublisher(ctx context.Context) {
	s.logger.Info("Scheduling publisher on a timer")

	t := ticker.NewTicker(time.Second*10, time.Second*10)
	defer t.Stop()

	for ; ; <-t.C {
		if err := s.runPublisher(ctx); err != nil {
			s.logger.Error("failed to run compliance publisher", zap.Error(err))
			continue
		}
	}
}
