package analytics

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	esSinkClient "github.com/opengovern/og-util/pkg/es/ingest/client"
	"github.com/opengovern/og-util/pkg/jq"
	integrationClient "github.com/opengovern/opengovernance/services/integration/client"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/opengovern/og-util/pkg/config"
	"github.com/opengovern/og-util/pkg/postgres"
	"github.com/opengovern/og-util/pkg/source"
	"github.com/opengovern/og-util/pkg/steampipe"
	workerConfig "github.com/opengovern/opengovernance/pkg/analytics/config"
	"github.com/opengovern/opengovernance/pkg/analytics/db"
	describeClient "github.com/opengovern/opengovernance/pkg/describe/client"
	inventoryClient "github.com/opengovern/opengovernance/pkg/inventory/client"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
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
				cmd.Context(),
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
	id                string
	jq                *jq.JobQueue
	config            workerConfig.WorkerConfig
	logger            *zap.Logger
	db                db.Database
	integrationClient integrationClient.IntegrationServiceClient
	schedulerClient   describeClient.SchedulerServiceClient
	inventoryClient   inventoryClient.InventoryServiceClient
	sinkClient        esSinkClient.EsSinkServiceClient
}

func NewWorker(
	id string,
	conf workerConfig.WorkerConfig,
	logger *zap.Logger,
	ctx context.Context,
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

	if err := jq.Stream(ctx, StreamName, "analytics job queue", []string{JobQueueTopic, JobResultQueueTopic}, 1000); err != nil {
		return nil, err
	}

	w.jq = jq

	w.config = conf
	w.logger = logger

	w.integrationClient = integrationClient.NewIntegrationServiceClient(conf.Integration.BaseURL)
	w.schedulerClient = describeClient.NewSchedulerServiceClient(conf.Scheduler.BaseURL)
	w.inventoryClient = inventoryClient.NewInventoryServiceClient(conf.Inventory.BaseURL)
	w.sinkClient = esSinkClient.NewEsSinkServiceClient(logger, conf.EsSink.BaseURL)
	return w, nil
}

func (w *Worker) Run(ctx context.Context) error {
	defer func() {
		if r := recover(); r != nil {
			w.logger.Error("panic happened with error", zap.Error(fmt.Errorf("%v", r)))
		}
	}()

	w.logger.Info("Starting analytics worker")

	if err := steampipe.PopulateSteampipeConfig(w.config.ElasticSearch, source.CloudAWS); err != nil {
		w.logger.Error("failed to populate steampipe config for aws plugin", zap.Error(err))
		return err
	}

	if err := steampipe.PopulateSteampipeConfig(w.config.ElasticSearch, source.CloudAzure); err != nil {
		w.logger.Error("failed to populate steampipe config for azure plugin", zap.Error(err))
		return err
	}

	if err := steampipe.PopulateOpenGovernancePluginSteampipeConfig(w.config.ElasticSearch, w.config.Steampipe); err != nil {
		w.logger.Error("failed to populate steampipe config for opengovernance plugin", zap.Error(err))
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

	consumeCtx, err := w.jq.Consume(ctx, "analytics-worker", StreamName, []string{JobQueueTopic}, consumerGroup, func(msg jetstream.Msg) {
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

		result := job.Do(w.jq, w.db, steampipeConn, w.integrationClient, w.schedulerClient, w.inventoryClient, w.sinkClient, w.logger, w.config, ctx)

		w.logger.Info("Job finished", zap.Uint("jobID", job.JobID))

		resultJson, err := json.Marshal(result)
		if err != nil {
			w.logger.Error("Failed to marshal result", zap.Error(err))
			return
		}

		if _, err := w.jq.Produce(ctx, JobResultQueueTopic, resultJson, fmt.Sprintf("job-result-%d", job.JobID)); err != nil {
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

	<-ctx.Done()
	consumeCtx.Drain()
	consumeCtx.Stop()

	return nil
}

func (w *Worker) Stop() {
}
