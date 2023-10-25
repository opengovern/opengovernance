package worker

import (
	"context"
	"encoding/json"
	kafka2 "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	complianceClient "github.com/kaytu-io/kaytu-engine/pkg/compliance/client"
	"github.com/kaytu-io/kaytu-util/pkg/config"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"go.uber.org/zap"
	"strings"
	"time"
)

const (
	JobQueue      = "compliance-worker-job-queue"
	ResultQueue   = "compliance-worker-job-result"
	ConsumerGroup = "compliance-worker"

	JobTimeoutCheckInterval = 5 * time.Minute
)

type Config struct {
	RabbitMQ              config.RabbitMQ
	ElasticSearch         config.ElasticSearch
	Kafka                 config.Kafka
	Compliance            config.KaytuService
	Inventory             config.KaytuService
	Onboard               config.KaytuService
	Scheduler             config.KaytuService
	PrometheusPushAddress string
}

type Worker struct {
	config        Config
	logger        *zap.Logger
	steampipeConn *steampipe.Database
	esClient      kaytu.Client
	kafkaProducer *kafka2.Producer
}

func InitializeNewWorker(
	config Config,
	logger *zap.Logger,
	prometheusPushAddress string,
) (*Worker, error) {
	steampipeConn, err := steampipe.StartSteampipeServiceAndGetConnection(logger)
	if err != nil {
		return nil, err
	}

	esClient, err := kaytu.NewClient(kaytu.ClientConfig{
		Addresses: []string{config.ElasticSearch.Address},
		Username:  &config.ElasticSearch.Username,
		Password:  &config.ElasticSearch.Password,
	})
	if err != nil {
		return nil, err
	}

	producer, err := newKafkaProducer(strings.Split(config.Kafka.Addresses, ","))
	if err != nil {
		return nil, err
	}

	w := &Worker{
		config:        config,
		logger:        logger,
		steampipeConn: steampipeConn,
		esClient:      esClient,
		kafkaProducer: producer,
	}

	return w, nil
}

func (w *Worker) Run() error {
	ctx := context.Background()
	consumer, err := kafka.NewTopicConsumer(ctx, strings.Split(w.config.Kafka.Addresses, ","), JobQueue, ConsumerGroup)
	if err != nil {
		return err
	}
	msgs := consumer.Consume(ctx)
	t := time.NewTicker(JobTimeoutCheckInterval)
	defer t.Stop()

	for {
		select {
		case msg := <-msgs:
			t.Reset(JobTimeoutCheckInterval)

			commit, requeue, err := w.ProcessMessage(msg)
			if err != nil {
				w.logger.Error("failed to process message", zap.Error(err))
			}

			if requeue {
				//TODO
			}

			if commit {
				err := consumer.Commit(msg)
				if err != nil {
					w.logger.Error("failed to commit message", zap.Error(err))
				}
			}
		case _ = <-t.C:
			return nil
		}
	}
}

func (w *Worker) ProcessMessage(msg *kafka2.Message) (commit bool, requeue bool, err error) {
	var job Job
	err = json.Unmarshal(msg.Value, &job)
	if err != nil {
		return true, false, err
	}

	w.logger.Info("running job", zap.String("job", string(msg.Value)))
	err = job.Run(JobConfig{
		config:           w.config,
		logger:           w.logger,
		complianceClient: complianceClient.NewComplianceClient(w.config.Compliance.BaseURL),
		steampipeConn:    w.steampipeConn,
		esClient:         w.esClient,
		kafkaProducer:    w.kafkaProducer,
	})
	if err != nil {
		return true, false, err
	}

	return true, false, nil
}

func (w *Worker) Stop() error {
	w.steampipeConn.Conn().Close()
	steampipe.StopSteampipeService(w.logger)

	return nil
}

func newKafkaProducer(kafkaServers []string) (*kafka2.Producer, error) {
	return kafka2.NewProducer(&kafka2.ConfigMap{
		"bootstrap.servers": strings.Join(kafkaServers, ","),
		"acks":              "all",
		"retries":           3,
		"linger.ms":         1,
		"batch.size":        1000000,
		"compression.type":  "lz4",
	})
}
