package analytics

import (
	"encoding/json"
	"errors"
	"fmt"
	confluent_kafka "github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/kaytu-io/kaytu-engine/pkg/analytics/db"
	describeClient "github.com/kaytu-io/kaytu-engine/pkg/describe/client"
	onboardClient "github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-util/pkg/config"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"github.com/kaytu-io/kaytu-util/pkg/queue"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"strings"
)

var (
	analyticsJobQueue       = "analytics-jobs-queue"
	analyticsJobResultQueue = "analytics-results-queue"
)

type WorkerConfig struct {
	RabbitMQ      config.RabbitMQ
	Kafka         config.Kafka
	PostgreSQL    config.Postgres
	ElasticSearch config.ElasticSearch
	Steampipe     config.Postgres
	Onboard       config.KaytuService
	Scheduler     config.KaytuService
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
	jobQueue        queue.Interface
	jobResultQueue  queue.Interface
	config          WorkerConfig
	logger          *zap.Logger
	db              db.Database
	steampipeDB     *steampipe.Database
	onboardClient   onboardClient.OnboardServiceClient
	schedulerClient describeClient.SchedulerServiceClient
	kfkProducer     *confluent_kafka.Producer
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

	err = steampipe.PopulateSteampipeConfig(conf.ElasticSearch, source.CloudAWS, "all", nil)
	if err != nil {
		logger.Error("failed to populate steampipe config for aws plugin", zap.Error(err))
		return nil, err
	}
	err = steampipe.PopulateSteampipeConfig(conf.ElasticSearch, source.CloudAzure, "all", nil)
	if err != nil {
		logger.Error("failed to populate steampipe config for azure plugin", zap.Error(err))
		return nil, err
	}
	err = steampipe.PopulateKaytuPluginSteampipeConfig(conf.ElasticSearch, conf.Steampipe, nil)
	if err != nil {
		logger.Error("failed to populate steampipe config for kaytu plugin", zap.Error(err))
		return nil, err
	}

	steampipeConn, err := steampipe.StartSteampipeServiceAndGetConnection(logger)
	if err != nil {
		return nil, err
	}
	w.steampipeDB = steampipeConn
	fmt.Println("Connected to the steampipe database: ", conf.Steampipe.DB)

	qCfg := queue.Config{}
	qCfg.Server.Username = conf.RabbitMQ.Username
	qCfg.Server.Password = conf.RabbitMQ.Password
	qCfg.Server.Host = conf.RabbitMQ.Service
	qCfg.Server.Port = 5672
	qCfg.Queue.Name = analyticsJobQueue
	qCfg.Queue.Durable = true
	qCfg.Consumer.ID = w.id
	reportJobQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	w.jobQueue = reportJobQueue

	qCfg = queue.Config{}
	qCfg.Server.Username = conf.RabbitMQ.Username
	qCfg.Server.Password = conf.RabbitMQ.Password
	qCfg.Server.Host = conf.RabbitMQ.Service
	qCfg.Server.Port = 5672
	qCfg.Queue.Name = analyticsJobResultQueue
	qCfg.Queue.Durable = true
	qCfg.Producer.ID = w.id
	resultQueue, err := queue.New(qCfg)
	if err != nil {
		return nil, err
	}

	w.jobResultQueue = resultQueue

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
	return w, nil
}

func (w *Worker) Run() error {
	defer func() {
		if r := recover(); r != nil {
			w.logger.Error("paniced with error", zap.Error(fmt.Errorf("%v", r)))
		}
	}()

	w.logger.Info("Starting analytics worker")

	msgs, err := w.jobQueue.Consume()
	if err != nil {
		w.logger.Info("Failed to consume due to error", zap.Error(err))
		return err
	}

	w.logger.Info("Reading message")
	msg := <-msgs

	w.logger.Info("Parsing job")

	var job Job
	if err := json.Unmarshal(msg.Body, &job); err != nil {
		w.logger.Error("Failed to unmarshal task", zap.Error(err))

		if err2 := msg.Nack(false, false); err2 != nil {
			w.logger.Error("Failed nacking message", zap.Error(err2))
		}

		return err
	}

	w.logger.Info("Running the job", zap.Uint("jobID", job.JobID))

	result := job.Do(w.db, w.steampipeDB, w.kfkProducer, w.config.Kafka.Topic, w.onboardClient, w.schedulerClient, w.logger)

	w.logger.Info("Job finished", zap.Uint("jobID", job.JobID))

	if err := w.jobResultQueue.Publish(result); err != nil {
		w.logger.Error("Failed to send results to queue", zap.Error(err))
	}

	w.logger.Info("A job is done and result is published into the result queue", zap.String("result", fmt.Sprintf("%v", result)))
	if err := msg.Ack(false); err != nil {
		w.logger.Error("Failed acking message", zap.Error(err))
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
