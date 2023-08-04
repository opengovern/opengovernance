package migrator

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"

	elasticsearchv7 "github.com/elastic/go-elasticsearch/v7"
	"github.com/kaytu-io/kaytu-engine/pkg/metadata/client"
	"github.com/kaytu-io/kaytu-engine/pkg/migrator/db"
	"github.com/kaytu-io/kaytu-engine/pkg/migrator/internal"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"github.com/kaytu-io/kaytu-util/pkg/queue"
	"github.com/prometheus/client_golang/prometheus/push"
	"go.uber.org/zap"
)

type Worker struct {
	db             db.Database
	elastic        elasticsearchv7.Config
	logger         *zap.Logger
	pusher         *push.Pusher
	metadataClient client.MetadataServiceClient

	jobQueue queue.Interface

	conf JobConfig
}

func InitializeWorker(
	conf JobConfig,
	logger *zap.Logger,
	prometheusPushAddress string,
) (w *Worker, err error) {
	w = &Worker{
		logger: logger,
	}
	defer func() {
		if err != nil && w != nil {
			w.Stop()
		}
	}()

	cfg := postgres.Config{
		Host:    conf.PostgreSQL.Host,
		Port:    conf.PostgreSQL.Port,
		User:    conf.PostgreSQL.Username,
		Passwd:  conf.PostgreSQL.Password,
		DB:      conf.PostgreSQL.DB,
		SSLMode: conf.PostgreSQL.SSLMode,
	}
	orm, err := postgres.NewClient(&cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("new postgres client: %w", err)
	}
	w.db = db.Database{ORM: orm}
	logger.Info("Connected to the postgres database", zap.String("database", conf.PostgreSQL.DB))

	w.pusher = push.New(prometheusPushAddress, "migrator")
	w.elastic = elasticsearchv7.Config{
		Addresses: []string{conf.ElasticSearch.Address},
		Username:  conf.ElasticSearch.Username,
		Password:  conf.ElasticSearch.Password,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	if err != nil {
		return nil, err
	}

	qCfg := queue.Config{}
	qCfg.Server.Username = conf.RabbitMqUsername
	qCfg.Server.Password = conf.RabbitMqPassword
	qCfg.Server.Host = conf.RabbitMqService
	qCfg.Server.Port = 5672
	qCfg.Queue.Name = conf.RabbitMqQueue
	qCfg.Queue.Durable = true
	qCfg.Consumer.ID = "migrator"
	jobQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}
	w.jobQueue = jobQueue

	w.metadataClient = client.NewMetadataServiceClient(conf.Metadata.BaseURL)

	w.conf = conf
	return w, nil
}

func (w *Worker) Run() error {
	defer func() {
		if r := recover(); r != nil {
			w.logger.Error("paniced with error", zap.Error(fmt.Errorf("%v", r)))
		}
	}()

	msgs, err := w.jobQueue.Consume()
	if err != nil {
		return err
	}
	msg := <-msgs

	w.logger.Info("Received a message", zap.String("message", string(msg.Body)))
	if err := NewJob(w.db, w.elastic, w.logger, w.pusher, w.metadataClient, w.conf).Run(); err != nil {
		w.logger.Error("Failed to handle message", zap.Error(err))
		err = msg.Nack(false, true)
		if err != nil {
			w.logger.Error("Failed to nack message", zap.Error(err))
		}

	}
	err = msg.Ack(false)
	if err != nil {
		w.logger.Error("Failed to ack message", zap.Error(err))

	}
	w.logger.Info("Message handled successfully", zap.String("message", string(msg.Body)))

	return nil
}

func (w *Worker) Stop() {
	w.jobQueue.Close()
	os.RemoveAll(internal.ComplianceGitPath)
	os.RemoveAll(internal.QueriesGitPath)
	os.RemoveAll(internal.InsightsGitPath)
	os.RemoveAll(internal.AnalyticsGitPath)
}
