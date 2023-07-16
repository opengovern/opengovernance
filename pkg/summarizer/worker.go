package summarizer

import (
	"encoding/json"
	"fmt"
	"strings"

	confluent_kafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/client"
	"github.com/kaytu-io/kaytu-engine/pkg/summarizer/api"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"github.com/kaytu-io/kaytu-util/pkg/queue"

	"github.com/kaytu-io/kaytu-engine/pkg/inventory"

	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"

	"github.com/prometheus/client_golang/prometheus/push"
	"go.uber.org/zap"
)

type JobType string

const MaxKafkaSendBatchSize = 10000

const (
	JobType_ResourceMustSummarizer JobType = "resourceMustSummarizer"
	JobType_ComplianceSummarizer   JobType = "complianceSummarizer"
)

type Worker struct {
	id               string
	jobQueue         queue.Interface
	jobResultQueue   queue.Interface
	kfkProducer      *confluent_kafka.Producer
	kfkTopic         string
	logger           *zap.Logger
	es               keibi.Client
	db               inventory.Database
	complianceClient client.ComplianceServiceClient
	pusher           *push.Pusher
}

type SummarizeJob struct {
	JobID   uint
	JobType JobType
}

type SummarizeJobResult struct {
	JobID   uint
	Status  api.SummarizerJobStatus
	Error   string
	JobType JobType
}

func InitializeWorker(
	id string,
	rabbitMQUsername string, rabbitMQPassword string, rabbitMQHost string, rabbitMQPort int,
	summarizerJobQueue string, summarizerJobResultQueue string,
	kafkaBrokers []string, kafkaTopic string,
	logger *zap.Logger,
	prometheusPushAddress string,
	complianceBaseUrl string,
	elasticSearchAddress string, elasticSearchUsername string, elasticSearchPassword string,
	postgresHost string, postgresPort string, postgresDb string, postgresUsername string, postgresPassword string, postgresSSLMode string,
) (w *Worker, err error) {
	if id == "" {
		return nil, fmt.Errorf("'id' must be set to a non empty string")
	} else if kafkaTopic == "" {
		return nil, fmt.Errorf("'kfkTopic' must be set to a non empty string")
	}

	w = &Worker{id: id, kfkTopic: kafkaTopic}
	defer func() {
		if err != nil && w != nil {
			w.Stop()
		}
	}()

	w.complianceClient = client.NewComplianceClient(complianceBaseUrl)

	qCfg := queue.Config{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = summarizerJobQueue
	qCfg.Queue.Durable = true
	qCfg.Consumer.ID = w.id
	summarizerQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	w.jobQueue = summarizerQueue

	qCfg = queue.Config{}
	qCfg.Server.Username = rabbitMQUsername
	qCfg.Server.Password = rabbitMQPassword
	qCfg.Server.Host = rabbitMQHost
	qCfg.Server.Port = rabbitMQPort
	qCfg.Queue.Name = summarizerJobResultQueue
	qCfg.Queue.Durable = true
	qCfg.Producer.ID = w.id
	summarizerResultsQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	w.jobResultQueue = summarizerResultsQueue

	producer, err := newKafkaProducer(kafkaBrokers)
	if err != nil {
		return nil, err
	}

	w.kfkProducer = producer

	w.logger = logger

	w.pusher = push.New(prometheusPushAddress, "summarizer-worker")
	w.pusher.
		Collector(DoResourceSummarizerJobsCount).
		Collector(DoResourceSummarizerJobsDuration).
		Collector(DoComplianceSummarizerJobsCount).
		Collector(DoComplianceSummarizerJobsDuration)

	defaultAccountID := "default"
	w.es, err = keibi.NewClient(keibi.ClientConfig{
		Addresses: []string{elasticSearchAddress},
		Username:  &elasticSearchUsername,
		Password:  &elasticSearchPassword,
		AccountID: &defaultAccountID,
	})
	if err != nil {
		return nil, err
	}

	// setup postgres connection
	cfg := postgres.Config{
		Host:    postgresHost,
		Port:    postgresPort,
		User:    postgresUsername,
		Passwd:  postgresPassword,
		DB:      postgresDb,
		SSLMode: postgresSSLMode,
	}
	orm, err := postgres.NewClient(&cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("new postgres client: %w", err)
	}

	w.db = inventory.NewDatabase(orm)
	fmt.Println("Connected to the postgres database: ", postgresDb)

	return w, nil
}

func (w *Worker) Run() error {
	defer func() {
		if r := recover(); r != nil {
			w.logger.Error("Paniced while running worker", zap.Error(fmt.Errorf("%v", r)))
		}
	}()

	w.logger.Info("Running summarizer")
	msgs, err := w.jobQueue.Consume()
	if err != nil {
		return err
	}

	w.logger.Info("Consuming")
	msg := <-msgs

	w.logger.Info("Took the job")
	var summarizeJob SummarizeJob
	if err := json.Unmarshal(msg.Body, &summarizeJob); err != nil {
		w.logger.Error("Failed to unmarshal task", zap.Error(err))
		err2 := msg.Nack(false, false)
		if err2 != nil {
			w.logger.Error("Failed nacking message", zap.Error(err))
		}
		return err
	}

	switch summarizeJob.JobType {
	case "":
		fallthrough
	case JobType_ResourceMustSummarizer:
		w.logger.Info("Processing job", zap.Int("jobID", int(summarizeJob.JobID)))
		result := summarizeJob.DoMustSummarizer(w.es, w.db, w.kfkProducer, w.kfkTopic, w.logger)
		w.logger.Info("Publishing job result", zap.Int("jobID", int(summarizeJob.JobID)), zap.String("status", string(result.Status)))
		err = w.jobResultQueue.Publish(result)
		if err != nil {
			w.logger.Error("Failed to send results to queue: %s", zap.Error(err))
		}
	case JobType_ComplianceSummarizer:
		w.logger.Info("Processing job", zap.Int("jobID", int(summarizeJob.JobID)))
		result := summarizeJob.DoComplianceSummarizer(w.es, w.complianceClient, w.kfkProducer, w.kfkTopic, w.logger)
		w.logger.Info("Publishing job result", zap.Int("jobID", int(summarizeJob.JobID)), zap.String("status", string(result.Status)))
		err = w.jobResultQueue.Publish(result)
		if err != nil {
			w.logger.Error("Failed to send results to queue: %s", zap.Error(err))
		}
	}

	if err := msg.Ack(false); err != nil {
		w.logger.Error("Failed acking message", zap.Error(err))
	}

	err = w.pusher.Push()
	if err != nil {
		w.logger.Error("Failed to push metrics", zap.Error(err))
	}

	return nil
}

func (w *Worker) Stop() {
	w.pusher.Push()

	if w.jobQueue != nil {
		w.jobQueue.Close() //nolint,gosec
		w.jobQueue = nil
	}

	if w.jobResultQueue != nil {
		w.jobResultQueue.Close() //nolint,gosec
		w.jobResultQueue = nil
	}

	if w.kfkProducer != nil {
		w.kfkProducer.Close() //nolint,gosec
		w.kfkProducer = nil
	}
}

func newKafkaProducer(brokers []string) (*confluent_kafka.Producer, error) {
	return confluent_kafka.NewProducer(&confluent_kafka.ConfigMap{
		"bootstrap.servers":            strings.Join(brokers, ","),
		"linger.ms":                    100,
		"compression.type":             "lz4",
		"message.timeout.ms":           10000,
		"queue.buffering.max.messages": 100000,
		"queue.buffering.max.kbytes":   100000000,
	})
}
