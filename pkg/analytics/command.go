package analytics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	confluent_kafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kaytu-io/kaytu-engine/pkg/analytics/db"
	describeClient "github.com/kaytu-io/kaytu-engine/pkg/describe/client"
	inventoryClient "github.com/kaytu-io/kaytu-engine/pkg/inventory/client"
	onboardClient "github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-util/pkg/config"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"strings"
)

const (
	JobQueueTopic       = "analytics-jobs-queue"
	JobResultQueueTopic = "analytics-results-queue"
	consumerGroup       = "analytics-worker"
)

type WorkerConfig struct {
	RabbitMQ      config.RabbitMQ
	Kafka         config.Kafka
	PostgreSQL    config.Postgres
	ElasticSearch config.ElasticSearch
	Steampipe     config.Postgres
	Onboard       config.KaytuService
	Scheduler     config.KaytuService
	Inventory     config.KaytuService
}

func WorkerCommand() *cobra.Command {
	var (
		id  string
		cnf WorkerConfig
	)
	config.ReadFromEnv(&cnf, nil)

	cmd := &cobra.Command{
		PreRunE: func(cmd *cobra.Command, args []string) error {
			switch {
			case id == "":
				return errors.New("missing required flag 'id'")
			default:
				return nil
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			logger, err := zap.NewProduction()
			if err != nil {
				return err
			}

			w, err := InitializeWorker(
				id,
				cnf,
				logger,
			)

			if err != nil {
				return err
			}

			defer w.Stop()

			return w.Run()
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "The worker id")

	return cmd
}

type Worker struct {
	id              string
	jobQueue        *kafka.TopicConsumer
	kfkProducer     *confluent_kafka.Producer
	config          WorkerConfig
	logger          *zap.Logger
	db              db.Database
	onboardClient   onboardClient.OnboardServiceClient
	schedulerClient describeClient.SchedulerServiceClient
	inventoryClient inventoryClient.InventoryServiceClient
}

func InitializeWorker(
	id string,
	conf WorkerConfig,
	logger *zap.Logger,
) (w *Worker, err error) {
	if id == "" {
		return nil, fmt.Errorf("'id' must be set to a non empty string")
	}

	w = &Worker{id: id}
	defer func() {
		if err != nil && w != nil {
			w.Stop()
		}
	}()

	// setup postgres connection
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

	w.db = db.NewDatabase(orm)
	fmt.Println("Connected to the postgres database: ", conf.PostgreSQL.DB)

	err = w.db.Initialize()
	if err != nil {
		return nil, err
	}
	fmt.Println("Initialized postgres database: ", conf.PostgreSQL.DB)

	consumer, err := kafka.NewTopicConsumer(context.Background(),
		strings.Split(conf.Kafka.Addresses, ","), JobQueueTopic, consumerGroup, false)
	if err != nil {
		logger.Error("Failed to create kafka consumer", zap.Error(err))
		return nil, err
	}
	w.jobQueue = consumer

	producer, err := kafka.NewDefaultKafkaProducer(strings.Split(conf.Kafka.Addresses, ","))
	if err != nil {
		return nil, err
	}
	w.kfkProducer = producer

	fmt.Println(strings.Split(conf.Kafka.Addresses, ","), w.config.Kafka.Topic)

	w.config = conf
	w.logger = logger

	w.onboardClient = onboardClient.NewOnboardServiceClient(conf.Onboard.BaseURL, nil)
	w.schedulerClient = describeClient.NewSchedulerServiceClient(conf.Scheduler.BaseURL)
	w.inventoryClient = inventoryClient.NewInventoryServiceClient(conf.Inventory.BaseURL)
	return w, nil
}

func (w *Worker) Run() error {
	defer func() {
		if r := recover(); r != nil {
			w.logger.Error("paniced with error", zap.Error(fmt.Errorf("%v", r)))
		}
	}()

	w.logger.Info("Starting analytics worker")

	msgs := w.jobQueue.Consume(context.TODO(), w.logger, 100)

	err := steampipe.PopulateSteampipeConfig(w.config.ElasticSearch, source.CloudAWS)
	if err != nil {
		w.logger.Error("failed to populate steampipe config for aws plugin", zap.Error(err))
		return err
	}
	err = steampipe.PopulateSteampipeConfig(w.config.ElasticSearch, source.CloudAzure)
	if err != nil {
		w.logger.Error("failed to populate steampipe config for azure plugin", zap.Error(err))
		return err
	}
	err = steampipe.PopulateKaytuPluginSteampipeConfig(w.config.ElasticSearch, w.config.Steampipe)
	if err != nil {
		w.logger.Error("failed to populate steampipe config for kaytu plugin", zap.Error(err))
		return err
	}

	steampipeConn, err := steampipe.StartSteampipeServiceAndGetConnection(w.logger)
	if err != nil {
		return err
	}
	w.logger.Info("Connected to the steampipe database")
	defer steampipeConn.Conn().Close()
	defer steampipe.StopSteampipeService(w.logger)

	w.logger.Info("Reading messages from the queue")

	for msg := range msgs {
		w.logger.Info("Parsing job")

		var job Job
		if err := json.Unmarshal(msg.Value, &job); err != nil {
			w.logger.Error("Failed to unmarshal task", zap.Error(err), zap.String("value", string(msg.Value)))

			err2 := w.jobQueue.Commit(msg)
			if err2 != nil {
				w.logger.Error("Failed to commit message", zap.Error(err))
			}

			return err
		}

		w.logger.Info("Running the job", zap.Uint("jobID", job.JobID))

		result := job.Do(w.db, steampipeConn, w.kfkProducer, w.config.Kafka.Topic, w.onboardClient, w.schedulerClient, w.inventoryClient, w.logger)

		w.logger.Info("Job finished", zap.Uint("jobID", job.JobID))

		resultJson, err := json.Marshal(result)
		if err != nil {
			w.logger.Error("Failed to marshal result", zap.Error(err))
			return err
		}

		err = kafka.SyncSendWithRetry(w.logger, w.kfkProducer, []*confluent_kafka.Message{
			{
				TopicPartition: confluent_kafka.TopicPartition{
					Topic:     utils.GetPointer(JobResultQueueTopic),
					Partition: confluent_kafka.PartitionAny,
				},
				Value: resultJson,
			},
		}, nil, 5)
		if err != nil {
			w.logger.Error("Failed to send job result", zap.Error(err))
			return err
		}

		w.logger.Info("A job is done and result is published into the result queue", zap.String("result", fmt.Sprintf("%v", result)))
		err = w.jobQueue.Commit(msg)
		if err != nil {
			w.logger.Error("Failed to commit message", zap.Error(err))
		}
	}

	return nil
}

func (w *Worker) Stop() {
	w.jobQueue.Close()
	w.kfkProducer.Close()
}
