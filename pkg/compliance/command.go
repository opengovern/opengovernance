package compliance

import (
	"context"
	"errors"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/worker"
	"os"

	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"
	"github.com/kaytu-io/kaytu-util/pkg/config"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	S3AccessKey    = os.Getenv("S3_ACCESS_KEY")
	S3AccessSecret = os.Getenv("S3_ACCESS_SECRET")
	S3Region       = os.Getenv("S3_REGION")
)

func WorkerCommand() *cobra.Command {
	var (
		id  string
		cnf worker.Config
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

			w, err := worker.InitializeNewWorker(
				cnf,
				logger,
				cnf.PrometheusPushAddress,
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

type OpenAI struct {
	Token string
}

type ServerConfig struct {
	ES         config.ElasticSearch
	PostgreSQL config.Postgres
	Scheduler  config.KaytuService
	Onboard    config.KaytuService
	Inventory  config.KaytuService
	OpenAI     OpenAI
	RabbitMq   config.RabbitMQ
	Http       config.HttpServer

	MigratorJobQueueName string `yaml:"migrator_job_queue_name"`
}

func ServerCommand() *cobra.Command {
	return &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			return startHttpServer(cmd.Context())
		},
	}
}

func startHttpServer(ctx context.Context) error {
	var conf ServerConfig
	config.ReadFromEnv(&conf, nil)

	logger, err := zap.NewProduction()
	if err != nil {
		return fmt.Errorf("new logger: %w", err)
	}

	handler, err := InitializeHttpHandler(conf,
		S3Region, S3AccessKey, S3AccessSecret,
		logger)
	if err != nil {
		return fmt.Errorf("init http handler: %w", err)
	}

	return httpserver.RegisterAndStart(logger, conf.Http.Address, handler)
}
