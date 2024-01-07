package analytics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	workerConfig "github.com/kaytu-io/kaytu-engine/pkg/analytics/config"
	"github.com/kaytu-io/kaytu-engine/pkg/analytics/db"
	describeClient "github.com/kaytu-io/kaytu-engine/pkg/describe/client"
	inventoryClient "github.com/kaytu-io/kaytu-engine/pkg/inventory/client"
	"github.com/kaytu-io/kaytu-engine/pkg/jq"
	onboardClient "github.com/kaytu-io/kaytu-engine/pkg/onboard/client"
	"github.com/kaytu-io/kaytu-util/pkg/config"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const (
	JobQueueTopic       = "analytics-jobs-queue"
	JobResultQueueTopic = "analytics-results-queue"
	consumerGroup       = "analytics-worker"
	StreamName          = "analytics-worker"
)

func WorkerCommand() *cobra.Command {
	var (
		id  string
		cnf workerConfig.WorkerConfig
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

			w, err := NewWorker(
				id,
				cnf,
				logger,
			)
			if err != nil {
				return err
			}

			defer w.Stop()

			return w.Run(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "The worker id")

	return cmd
}

type Worker struct {
	id              string
	jq              *jq.JobQueue
	config          workerConfig.WorkerConfig
	logger          *zap.Logger
	db              db.Database
	onboardClient   onboardClient.OnboardServiceClient
	schedulerClient describeClient.SchedulerServiceClient
	inventoryClient inventoryClient.InventoryServiceClient
}

func NewWorker(
	id string,
	conf workerConfig.WorkerConfig,
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

	jq, err := jq.New(conf.NATS.URL, logger)
	if err != nil {
		return nil, err
	}
	w.jq = jq

	w.config = conf
	w.logger = logger

	w.onboardClient = onboardClient.NewOnboardServiceClient(conf.Onboard.BaseURL)
	w.schedulerClient = describeClient.NewSchedulerServiceClient(conf.Scheduler.BaseURL)
	w.inventoryClient = inventoryClient.NewInventoryServiceClient(conf.Inventory.BaseURL)
	return w, nil
}

func (w *Worker) Run(ctx context.Context) error {
	defer func() {
		if r := recover(); r != nil {
			w.logger.Error("panic happened with error", zap.Error(fmt.Errorf("%v", r)))
		}
	}()

	if err := w.jq.Stream(context.Background(), StreamName, "analytics jobs", []string{JobQueueTopic, JobResultQueueTopic}); err != nil {
		return err
	}

	w.logger.Info("Starting analytics worker")

	if err := steampipe.PopulateSteampipeConfig(w.config.ElasticSearch, source.CloudAWS); err != nil {
		w.logger.Error("failed to populate steampipe config for aws plugin", zap.Error(err))
		return err
	}

	if err := steampipe.PopulateSteampipeConfig(w.config.ElasticSearch, source.CloudAzure); err != nil {
		w.logger.Error("failed to populate steampipe config for azure plugin", zap.Error(err))
		return err
	}

	if err := steampipe.PopulateKaytuPluginSteampipeConfig(w.config.ElasticSearch, w.config.Steampipe); err != nil {
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

	consumeCtx, err := w.jq.Consume(context.Background(), "analytics-worker", StreamName, []string{JobQueueTopic}, consumerGroup, func(msg jetstream.Msg) {
		w.logger.Info("Parsing job")

		var job Job
		if err := json.Unmarshal(msg.Data(), &job); err != nil {
			w.logger.Error("Failed to unmarshal task", zap.Error(err), zap.ByteString("value", msg.Data()))

			if err := msg.Ack(); err != nil {
				w.logger.Error("Failed to commit message", zap.Error(err))
			}

			return
		}

		w.logger.Info("Running the job", zap.Uint("id", job.JobID))

		result := job.Do(w.jq, w.db, steampipeConn, w.onboardClient, w.schedulerClient, w.inventoryClient, w.logger, w.config)

		w.logger.Info("Job finished", zap.Uint("jobID", job.JobID))

		resultJson, err := json.Marshal(result)
		if err != nil {
			w.logger.Error("Failed to marshal result", zap.Error(err))
			return
		}

		if err := w.jq.Produce(context.Background(), JobQueueTopic, resultJson, fmt.Sprintf("job-%d", job.JobID)); err != nil {
			w.logger.Error("Failed to send job result", zap.Error(err))
			return
		}

		w.logger.Info("A job is done and result is published into the result queue", zap.String("result", fmt.Sprintf("%v", result)))
		if err := msg.Ack(); err != nil {
			w.logger.Error("Failed to commit message", zap.Error(err))
		}
	})
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			consumeCtx.Stop()
		default:
			continue
		}
	}
}

func (w *Worker) Stop() {
}
