package service

import (
	"context"
	"encoding/json"
	"github.com/kaytu-io/kaytu-engine/pkg/jq"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-util/pkg/es"
	essdk "github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/labstack/echo/v4"
	"net/http"

	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
	"time"
)

const (
	StreamName     = "es-sink"
	SinkQueueTopic = "es-sink-queue"
	ConsumerGroup  = "es-sink-consumer"
)

type EsSinkService struct {
	logger        *zap.Logger
	elasticSearch essdk.Client
	nats          *jq.JobQueue
	esSinkModule  *EsSinkModule
}

func NewEsSinkService(ctx context.Context, logger *zap.Logger, elasticSearch essdk.Client, nats *jq.JobQueue) (*EsSinkService, error) {
	service := EsSinkService{
		logger:        logger,
		elasticSearch: elasticSearch,
		nats:          nats,
	}

	esSinkModule, err := NewEsSinkModule(ctx, logger, elasticSearch)
	if err != nil {
		logger.Error("failed to create es sink module", zap.Error(err))
		return nil, err
	}
	service.esSinkModule = esSinkModule

	err = service.nats.StreamWithConfig(ctx, StreamName, "es sink stream", []string{SinkQueueTopic}, jetstream.StreamConfig{
		//Name:                 "",
		//Description:          "",
		//Subjects:             nil,
		Retention:    jetstream.WorkQueuePolicy,
		MaxConsumers: -1,
		MaxMsgs:      15000000,
		MaxBytes:     1024 * 15000000,
		Discard:      jetstream.DiscardNew,
		MaxAge:       time.Hour * 48,
		MaxMsgSize:   50 * 1024 * 1024,
		Storage:      jetstream.FileStorage,
		Replicas:     1,
		Duplicates:   15 * time.Minute,
		Compression:  jetstream.S2Compression,
		ConsumerLimits: jetstream.StreamConsumerLimits{
			MaxAckPending: 1000,
		},
	})
	if err != nil {
		return nil, err
	}

	return &service, nil
}

func (s *EsSinkService) Start(ctx context.Context) {
	s.logger.Info("starting es sink service")
	defer s.logger.Info("es sink service stopped")

	utils.EnsureRunGoroutine(func() {
		s.ConsumeCycle(ctx)
	})

	s.esSinkModule.Start(ctx)
}

func (s *EsSinkService) ConsumeCycle(ctx context.Context) {
	consumeCtx, err := s.nats.Consume(ctx, "es-sink", StreamName, []string{SinkQueueTopic}, ConsumerGroup, func(msg jetstream.Msg) {
		var doc es.Doc
		err := json.Unmarshal(msg.Data(), &doc)
		if err != nil {
			s.logger.Error("failed to unmarshal doc", zap.Error(err), zap.Any("msg", msg))
			return
		}

		s.esSinkModule.QueueDoc(doc)

		err = msg.Ack()
		if err != nil {
			s.logger.Error("failed to ack message", zap.Error(err), zap.Any("msg", msg))
		}
	})
	if err != nil {
		s.logger.Fatal("failed to consume", zap.Error(err))
	}

	s.logger.Info("consuming", zap.String("stream", StreamName), zap.String("topic", SinkQueueTopic))

	<-ctx.Done()
	consumeCtx.Drain()
	consumeCtx.Stop()
}

func (s *EsSinkService) Ingest(ctx context.Context, docs []es.Doc) error {
	failedCount := 0
	for _, doc := range docs {
		keys, _ := doc.KeysAndIndex()
		docJson, err := json.Marshal(doc)
		if err != nil {
			s.logger.Error("failed to marshal doc", zap.Error(err))
			return err
		}
		err = s.nats.Produce(ctx, SinkQueueTopic, docJson, es.HashOf(keys...))
		if err != nil {
			s.logger.Error("failed to produce message", zap.Error(err), zap.Any("doc", doc))
			failedCount++
		}
	}
	if failedCount > len(docs)/2 {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to produce messages")
	}

	return nil
}
