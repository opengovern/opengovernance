package jq

import (
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.uber.org/zap"
)

type JobQueue struct {
	conn   *nats.Conn
	js     *jetstream.JetStream
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
