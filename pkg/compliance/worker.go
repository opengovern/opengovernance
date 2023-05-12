package compliance

import (
	"encoding/json"
	"fmt"
	"strings"

	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/worker"
	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"

	client2 "gitlab.com/keibiengine/keibi-engine/pkg/compliance/client"

	"gitlab.com/keibiengine/keibi-engine/pkg/onboard/client"

	"github.com/prometheus/client_golang/prometheus/push"

	"gitlab.com/keibiengine/keibi-engine/pkg/internal/queue"
	"go.uber.org/zap"
	"gopkg.in/Shopify/sarama.v1"
)

type Worker struct {
	id               string
	jobQueue         queue.Interface
	jobResultQueue   queue.Interface
	config           WorkerConfig
	kfkProducer      sarama.SyncProducer
	kfkTopic         string
	logger           *zap.Logger
	pusher           *push.Pusher
	onboardClient    client.OnboardServiceClient
	complianceClient client2.ComplianceServiceClient
	es               keibi.Client
}

func InitializeWorker(
	id string,
	config WorkerConfig,
	complianceReportJobQueue, complianceReportJobResultQueue string,
	logger *zap.Logger,
	prometheusPushAddress string,
) (w *Worker, err error) {
	if id == "" {
		return nil, fmt.Errorf("'id' must be set to a non empty string")
	}

	w = &Worker{id: id, kfkTopic: config.Kafka.Topic}
	defer func() {
		if err != nil && w != nil {
			w.Stop()
		}
	}()

	qCfg := queue.Config{}
	qCfg.Server.Username = config.RabbitMQ.Username
	qCfg.Server.Password = config.RabbitMQ.Password
	qCfg.Server.Host = config.RabbitMQ.Service
	qCfg.Server.Port = 5672
	qCfg.Queue.Name = complianceReportJobQueue
	qCfg.Queue.Durable = true
	qCfg.Consumer.ID = w.id
	reportJobQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	w.jobQueue = reportJobQueue

	qCfg = queue.Config{}
	qCfg.Server.Username = config.RabbitMQ.Username
	qCfg.Server.Password = config.RabbitMQ.Password
	qCfg.Server.Host = config.RabbitMQ.Service
	qCfg.Server.Port = 5672
	qCfg.Queue.Name = complianceReportJobResultQueue
	qCfg.Queue.Durable = true
	qCfg.Producer.ID = w.id
	reportResultQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	w.jobResultQueue = reportResultQueue

	producer, err := newKafkaProducer(strings.Split(config.Kafka.Addresses, ","))
	if err != nil {
		return nil, err
	}
	w.kfkProducer = producer
	w.config = config
	w.logger = logger

	w.onboardClient = client.NewOnboardServiceClient(config.Onboard.BaseURL, nil)
	w.complianceClient = client2.NewComplianceClient(config.Compliance.BaseURL)

	w.pusher = push.New(prometheusPushAddress, "compliance-report")

	defaultAccountID := "default"
	w.es, err = keibi.NewClient(keibi.ClientConfig{
		Addresses: []string{config.ElasticSearch.Address},
		Username:  &config.ElasticSearch.Username,
		Password:  &config.ElasticSearch.Password,
		AccountID: &defaultAccountID,
	})
	if err != nil {
		return nil, err
	}

	return w, nil
}

func (w *Worker) Run() error {
	defer func() {
		if r := recover(); r != nil {
			w.logger.Error("paniced with error", zap.Error(fmt.Errorf("%v", r)))
		}
	}()

	w.logger.Info("Starting compliance worker")

	msgs, err := w.jobQueue.Consume()
	if err != nil {
		w.logger.Info("Failed to consume due to error", zap.Error(err))
		return err
	}

	w.logger.Info("Reading message")
	msg := <-msgs

	w.logger.Info("Parsing job")

	var job worker.Job
	if err := json.Unmarshal(msg.Body, &job); err != nil {
		w.logger.Error("Failed to unmarshal task", zap.Error(err))

		if err2 := msg.Nack(false, false); err2 != nil {
			w.logger.Error("Failed nacking message", zap.Error(err2))
		}

		return err
	}

	w.logger.Info("Running the job", zap.Uint("jobID", job.JobID))

	result := job.Do(w.complianceClient, w.onboardClient, w.config.ElasticSearch, w.kfkProducer, w.kfkTopic, w.logger)

	w.logger.Info("Job finished", zap.Uint("jobID", job.JobID))

	if err := w.jobResultQueue.Publish(result); err != nil {
		w.logger.Error("Failed to send results to queue", zap.Error(err))
	}

	w.logger.Info("A job is done and result is published into the result queue", zap.String("result", fmt.Sprintf("%v", result)))
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
	if w.jobQueue != nil {
		w.jobQueue.Close()
		w.jobQueue = nil
	}

	if w.jobResultQueue != nil {
		w.jobResultQueue.Close()
		w.jobResultQueue = nil
	}
}
func newKafkaProducer(kafkaServers []string) (sarama.SyncProducer, error) {
	cfg := sarama.NewConfig()
	cfg.Producer.Retry.Max = 3
	cfg.Producer.RequiredAcks = sarama.WaitForAll
	cfg.Producer.Return.Successes = true
	cfg.Version = sarama.V2_1_0_0

	producer, err := sarama.NewSyncProducer(kafkaServers, cfg)
	if err != nil {
		return nil, err
	}

	return producer, nil
}
