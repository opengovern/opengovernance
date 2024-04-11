package service

import (
	"context"
	"github.com/kaytu-io/kaytu-engine/pkg/jq"
	es "github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
	"time"
)

const (
	StreamName     = "es-sink"
	SinkQueueTopic = "es-sink-queue"
)

type EsSinkService struct {
	logger        *zap.Logger
	elasticSearch es.Client
	nats          *jq.JobQueue
}

func NewEsSinkService(ctx context.Context, logger *zap.Logger, elasticSearch es.Client, nats *jq.JobQueue) (*EsSinkService, error) {
	service := EsSinkService{
		logger:        logger,
		elasticSearch: elasticSearch,
		nats:          nats,
	}

	err := service.nats.StreamWithConfig(ctx, StreamName, "es sink stream", []string{SinkQueueTopic}, jetstream.StreamConfig{
		//Name:                 "",
		//Description:          "",
		//Subjects:             nil,
		Retention:    jetstream.WorkQueuePolicy,
		MaxConsumers: -1,
		MaxMsgs:      15000000,
		MaxBytes:     1024 * 1024 * 15000000,
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

}
