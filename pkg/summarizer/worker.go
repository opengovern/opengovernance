package summarizer

import (
	"encoding/json"
	"fmt"

	"gitlab.com/keibiengine/keibi-engine/pkg/internal/postgres"
	"gitlab.com/keibiengine/keibi-engine/pkg/inventory"

	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"

	"github.com/prometheus/client_golang/prometheus/push"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/queue"
	"go.uber.org/zap"
	"gopkg.in/Shopify/sarama.v1"
)

type JobType string

const (
	JobType_ResourceSummarizer   JobType = "resourceSummarizer"
	JobType_ComplianceSummarizer JobType = "complianceSummarizer"
)

type Worker struct {
	id             string
	jobQueue       queue.Interface
	jobResultQueue queue.Interface
	kfkProducer    sarama.SyncProducer
	kfkTopic       string
	logger         *zap.Logger
	es             keibi.Client
	db             inventory.Database
	pusher         *push.Pusher
}

func InitializeWorker(
	id string,
	rabbitMQUsername string,
	rabbitMQPassword string,
	rabbitMQHost string,
	rabbitMQPort int,
	summarizerJobQueue string,
	summarizerJobResultQueue string,
	kafkaBrokers []string,
	kafkaTopic string,
	logger *zap.Logger,
	prometheusPushAddress string,
	elasticSearchAddress string,
	elasticSearchUsername string,
	elasticSearchPassword string,
	postgresHost string,
	postgresPort string,
	postgresDb string,
	postgresUsername string,
	postgresPassword string,
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
	w.pusher.Collector(DoResourceSummarizerJobsCount).
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
		Host:   postgresHost,
		Port:   postgresPort,
		User:   postgresUsername,
		Passwd: postgresPassword,
		DB:     postgresDb,
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
	var resourceJob ResourceJob
	if err := json.Unmarshal(msg.Body, &resourceJob); err != nil {
		w.logger.Error("Failed to unmarshal task", zap.Error(err))
		err2 := msg.Nack(false, false)
		if err2 != nil {
			w.logger.Error("Failed nacking message", zap.Error(err))
		}
		return err
	}

	if resourceJob.JobType == "" || resourceJob.JobType == JobType_ResourceSummarizer {
		w.logger.Info("Processing job", zap.Int("jobID", int(resourceJob.JobID)))
		result := resourceJob.Do(w.es, w.db, w.kfkProducer, w.kfkTopic, w.logger)
		w.logger.Info("Publishing job result", zap.Int("jobID", int(resourceJob.JobID)), zap.String("status", string(result.Status)))
		err = w.jobResultQueue.Publish(result)
		if err != nil {
			w.logger.Error("Failed to send results to queue: %s", zap.Error(err))
		}
	} else {
		var complianceJob ComplianceJob
		if err := json.Unmarshal(msg.Body, &complianceJob); err != nil {
			w.logger.Error("Failed to unmarshal task", zap.Error(err))
			err2 := msg.Nack(false, false)
			if err2 != nil {
				w.logger.Error("Failed nacking message", zap.Error(err))
			}
			return err
		}

		w.logger.Info("Processing job", zap.Int("jobID", int(complianceJob.JobID)))
		result := complianceJob.Do(w.es, w.kfkProducer, w.kfkTopic, w.logger)
		w.logger.Info("Publishing job result", zap.Int("jobID", int(complianceJob.JobID)), zap.String("status", string(result.Status)))
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

func newKafkaProducer(brokers []string) (sarama.SyncProducer, error) {
	cfg := sarama.NewConfig()
	cfg.Producer.Retry.Max = 3
	cfg.Producer.RequiredAcks = sarama.WaitForAll
	cfg.Producer.Return.Successes = true
	cfg.Version = sarama.V2_1_0_0

	producer, err := sarama.NewSyncProducer(brokers, cfg)
	if err != nil {
		return nil, err
	}

	return producer, nil
}
