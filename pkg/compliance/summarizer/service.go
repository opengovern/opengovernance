package summarizer

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"
	"time"

	kafka2 "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/summarizer/types"
	inventoryClient "github.com/kaytu-io/kaytu-engine/pkg/inventory/client"
	"github.com/kaytu-io/kaytu-engine/pkg/jq"
	onboardClient "github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-util/pkg/config"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"go.uber.org/zap"
)

const (
	JobQueueTopic    = "compliance-summarizer-job-queue"
	ResultQueueTopic = "compliance-summarizer-job-result"
	ConsumerGroup    = "compliance-summarizer"
)

type Config struct {
	ElasticSearch         config.ElasticSearch
	NATS                  config.NATS
	PrometheusPushAddress string
	Inventory             config.KaytuService
	Onboard               config.KaytuService
}

type Worker struct {
	config   Config
	logger   *zap.Logger
	esClient kaytu.Client
	jq       *jq.JobQueue

	inventoryClient inventoryClient.InventoryServiceClient
	onboardClient   onboardClient.OnboardServiceClient
}

func InitializeNewWorker(
	config Config,
	logger *zap.Logger,
	prometheusPushAddress string,
) (*Worker, error) {
	esClient, err := kaytu.NewClient(kaytu.ClientConfig{
		Addresses:     []string{config.ElasticSearch.Address},
		Username:      &config.ElasticSearch.Username,
		Password:      &config.ElasticSearch.Password,
		IsOpenSearch:  &config.ElasticSearch.IsOpenSearch,
		AwsRegion:     &config.ElasticSearch.AwsRegion,
		AssumeRoleArn: &config.ElasticSearch.AssumeRoleArn,
	})
	if err != nil {
		return nil, err
	}

	jq, err := jq.New(config.NATS.URL, logger)
	if err != nil {
		return nil, err
	}

	w := &Worker{
		config:          config,
		logger:          logger,
		esClient:        esClient,
		jq:              jq,
		inventoryClient: inventoryClient.NewInventoryServiceClient(config.Inventory.BaseURL),
		onboardClient:   onboardClient.NewOnboardServiceClient(config.Onboard.BaseURL),
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

	var job types.Job
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
	runtime.GC()
	w.logger.Info("running job", zap.String("job", string(msg.Value)))
	err = w.RunJob(job)
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
