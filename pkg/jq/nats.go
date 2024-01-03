package jq

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
)

type JobQueue struct {
	conn   *nats.Conn
	js     jetstream.JetStream
	logger *zap.Logger
}

func New(url string, logger *zap.Logger) (*JobQueue, error) {
	jq := &JobQueue{
		conn:   nil,
		js:     nil,
		logger: logger.Named("jq"),
	}

	conn, err := nats.Connect(
		url,
		nats.ReconnectHandler(jq.reconnectHandler),
		nats.DisconnectErrHandler(jq.disconnectHandler),
	)
	if err != nil {
		return nil, err
	}

	jq.conn = conn

	js, err := jetstream.New(conn)
	if err != nil {
		return nil, err
	}

	jq.js = js

	return jq, nil
}

func (jq *JobQueue) reconnectHandler(nc *nats.Conn) {
	jq.logger.Info("got reconnected", zap.String("url", nc.ConnectedUrl()))
}

func (jq *JobQueue) disconnectHandler(_ *nats.Conn, err error) {
	jq.logger.Error("got disconnected", zap.Error(err))
}

func (jq *JobQueue) closeHandler(nc *nats.Conn) {
	jq.logger.Fatal("connection lost", zap.Error(nc.LastError()))
}

// Consume consumes messages from the given topic using the specified queue group.
// it creates pull consumer which is the only mode that is available in the new version
// of nats.go library.
func (jq *JobQueue) Consume(
	ctx context.Context,
	service string,
	stream string,
	topics []string,
	queue string,
	handler func(jetstream.Msg),
) (jetstream.ConsumeContext, error) {
	consumer, err := jq.js.CreateOrUpdateConsumer(ctx, stream, jetstream.ConsumerConfig{
		Name:              fmt.Sprintf("%s-service", service),
		Durable:           "",
		Description:       fmt.Sprintf("%s Service", strings.ToTitle(service)),
		Replicas:          1,
		FilterSubjects:    topics,
		AckPolicy:         jetstream.AckExplicitPolicy,
		DeliverPolicy:     jetstream.DeliverAllPolicy,
		MaxAckPending:     -1,
		InactiveThreshold: time.Hour,
	})
	if err != nil {
		return nil, err
	}

	consumeCtx, err := consumer.Consume(handler)
	if err != nil {
		return nil, err
	}

	return consumeCtx, nil
}

func (jq *JobQueue) Produce(ctx context.Context, topic string, data []byte, id string) error {
	if _, err := jq.js.Publish(ctx, topic, data, jetstream.WithMsgID(id)); err != nil {
		return err
	}

	return nil
}
