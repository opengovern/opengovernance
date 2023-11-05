package summarizer

import (
	"context"
	"encoding/json"
	"fmt"
	kafka2 "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kaytu-io/kaytu-util/pkg/config"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"go.uber.org/zap"
	"strings"
	"time"
)

const (
	JobQueue      = "compliance-summarizer-job-queue"
	ResultQueue   = "compliance-summarizer-job-result"
	ConsumerGroup = "compliance-summarizer"

	JobTimeoutCheckInterval = 5 * time.Minute
)

type Config struct {
	ElasticSearch         config.ElasticSearch
	Kafka                 config.Kafka
	PrometheusPushAddress string
}

type Worker struct {
	config        Config
	logger        *zap.Logger
	esClient      kaytu.Client
	kafkaProducer *kafka2.Producer
}

func InitializeNewWorker(
	config Config,
	logger *zap.Logger,
	prometheusPushAddress string,
) (*Worker, error) {
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
		esClient:      esClient,
		kafkaProducer: producer,
	}

	return w, nil
}

func (w *Worker) Run() error {
	w.logger.Info("starting")

	ctx := context.Background()
	consumer, err := kafka.NewTopicConsumer(ctx, strings.Split(w.config.Kafka.Addresses, ","), JobQueue, ConsumerGroup, true)
	if err != nil {
		return err
	}
	msgs := consumer.Consume(ctx, w.logger, 1)
	t := time.NewTicker(JobTimeoutCheckInterval)
	defer t.Stop()

	w.logger.Info("starting to consume")
	for {
		select {
		case msg, ok := <-msgs:
			if !ok {
				return fmt.Errorf("tasks channel is closed")
			}
			w.logger.Info("received a job")
			t.Reset(JobTimeoutCheckInterval)

			err := w.ProcessMessage(msg)
			if err != nil {
				w.logger.Error("failed to process message", zap.Error(err))
			}
		case _ = <-t.C:
			w.logger.Info("still waiting for a job")
			continue
		}
	}
}

func (w *Worker) ProcessMessage(msg *kafka2.Message) error {
	startTime := time.Now()

	var job Job
	err := json.Unmarshal(msg.Value, &job)
	if err != nil {
		return err
	}

	defer func() {
		result := JobResult{
			Job:       job,
			StartedAt: startTime,
			Status:    ComplianceSummarizerSucceeded,
			Error:     "",
		}

		if err != nil {
			result.Error = err.Error()
			result.Status = ComplianceSummarizerFailed
		}

		resultJson, err := json.Marshal(result)
		if err != nil {
			w.logger.Error("failed to create job result json", zap.Error(err))
			return
		}

		resultMsg := kafka.Msg(fmt.Sprintf("job-result-%d", job.ID), resultJson, "", ResultQueue, kafka2.PartitionAny)
		_, err = kafka.SyncSend(w.logger, w.kafkaProducer, []*kafka2.Message{resultMsg}, nil)
		if err != nil {
			w.logger.Error("failed to publish job result", zap.String("jobResult", string(resultJson)), zap.Error(err))
		}
	}()

	w.logger.Info("running job", zap.String("job", string(msg.Value)))
	err = job.Run(JobConfig{
		config:        w.config,
		logger:        w.logger,
		esClient:      w.esClient,
		kafkaProducer: w.kafkaProducer,
	})
	if err != nil {
		w.logger.Info("failure while running job", zap.Error(err))
		return err
	}

	return nil
}

func (w *Worker) Stop() error {
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
