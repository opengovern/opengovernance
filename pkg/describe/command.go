package describe

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const (
	DescribeJobsQueueName                = "describe-jobs-queue"
	DescribeResultsQueueName             = "describe-results-queue"
	DescribeCleanupJobsQueueName         = "describe-cleanup-jobs-queue"
	ComplianceReportJobsQueueName        = "compliance-report-jobs-queue"
	ComplianceReportResultsQueueName     = "compliance-report-results-queue"
	ComplianceReportCleanupJobsQueueName = "compliance-report-cleanup-jobs-queue"
	InsightJobsQueueName                 = "insight-jobs-queue"
	InsightResultsQueueName              = "insight-results-queue"
	SummarizerJobsQueueName              = "summarizer-jobs-queue"
	SummarizerResultsQueueName           = "summarizer-results-queue"
	SourceEventsQueueName                = "source-events-queue"
	DescribeConnectionJobsQueueName      = "describe-connection-jobs-queue"
	DescribeConnectionResultsQueueName   = "describe-connection-results-queue"
)

var (
	RabbitMQService  = os.Getenv("RABBITMQ_SERVICE")
	RabbitMQPort     = 5672
	RabbitMQUsername = os.Getenv("RABBITMQ_USERNAME")
	RabbitMQPassword = os.Getenv("RABBITMQ_PASSWORD")

	KafkaService = os.Getenv("KAFKA_SERVICE")

	PostgreSQLHost     = os.Getenv("POSTGRESQL_HOST")
	PostgreSQLPort     = os.Getenv("POSTGRESQL_PORT")
	PostgreSQLDb       = os.Getenv("POSTGRESQL_DB")
	PostgreSQLUser     = os.Getenv("POSTGRESQL_USERNAME")
	PostgreSQLPassword = os.Getenv("POSTGRESQL_PASSWORD")
	PostgreSQLSSLMode  = os.Getenv("POSTGRESQL_SSLMODE")

	VaultAddress  = os.Getenv("VAULT_ADDRESS")
	VaultToken    = os.Getenv("VAULT_TOKEN")
	VaultRoleName = os.Getenv("VAULT_ROLE")
	VaultCaPath   = os.Getenv("VAULT_TLS_CA_PATH")
	VaultUseTLS   = strings.ToLower(strings.TrimSpace(os.Getenv("VAULT_USE_TLS"))) == "true"

	ElasticSearchAddress  = os.Getenv("ES_ADDRESS")
	ElasticSearchUsername = os.Getenv("ES_USERNAME")
	ElasticSearchPassword = os.Getenv("ES_PASSWORD")

	HttpServerAddress = os.Getenv("HTTP_ADDRESS")

	PrometheusPushAddress = os.Getenv("PROMETHEUS_PUSH_ADDRESS")

	RedisAddress  = os.Getenv("REDIS_ADDRESS")
	CacheAddress  = os.Getenv("CACHE_ADDRESS")
	JaegerAddress = os.Getenv("JAEGER_ADDRESS")

	DescribeIntervalHours   = os.Getenv("DESCRIBE_INTERVAL_HOURS")
	ComplianceIntervalHours = os.Getenv("COMPLIANCE_INTERVAL_HOURS")
	InsightIntervalHours    = os.Getenv("INSIGHT_INTERVAL_HOURS")
	CurrentWorkspaceID      = os.Getenv("CURRENT_NAMESPACE")
	WorkspaceBaseURL        = os.Getenv("WORKSPACE_BASE_URL")
	ComplianceBaseURL       = os.Getenv("COMPLIANCE_BASE_URL")

	AccountConcurrentDescribe = os.Getenv("ACCOUNT_CONCURRENT_DESCRIBE")
)

func SchedulerCommand() *cobra.Command {
	var (
		id string
	)
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
			s, err := InitializeScheduler(
				id,
				RabbitMQUsername,
				RabbitMQPassword,
				RabbitMQService,
				RabbitMQPort,
				DescribeJobsQueueName,
				DescribeResultsQueueName,
				DescribeConnectionJobsQueueName,
				DescribeConnectionResultsQueueName,
				DescribeCleanupJobsQueueName,
				ComplianceReportJobsQueueName,
				ComplianceReportResultsQueueName,
				ComplianceReportCleanupJobsQueueName,
				InsightJobsQueueName,
				InsightResultsQueueName,
				SummarizerJobsQueueName,
				SummarizerResultsQueueName,
				SourceEventsQueueName,
				PostgreSQLUser,
				PostgreSQLPassword,
				PostgreSQLHost,
				PostgreSQLPort,
				PostgreSQLDb,
				PostgreSQLSSLMode,
				HttpServerAddress,
				DescribeIntervalHours,
				ComplianceIntervalHours,
				InsightIntervalHours,
			)
			if err != nil {
				return err
			}

			defer s.Stop()

			return s.Run()
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "The scheduler id")

	return cmd
}

func WorkerCommand() *cobra.Command {
	var (
		id             string
		resourcesTopic string
	)
	cmd := &cobra.Command{
		PreRunE: func(cmd *cobra.Command, args []string) error {
			switch {
			case id == "":
				return errors.New("missing required flag 'id'")
			case resourcesTopic == "":
				return errors.New("missing required flag 'resources-topic'")
			default:
				return nil
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, err := zap.NewProduction()
			if err != nil {
				return err
			}

			cmd.SilenceUsage = true

			w, err := InitializeWorker(
				id,
				RabbitMQUsername,
				RabbitMQPassword,
				RabbitMQService,
				RabbitMQPort,
				DescribeJobsQueueName,
				DescribeResultsQueueName,
				strings.Split(KafkaService, ","),
				resourcesTopic,
				VaultAddress,
				VaultRoleName,
				VaultToken,
				VaultCaPath,
				VaultUseTLS,
				logger,
				PrometheusPushAddress,
				RedisAddress,
				JaegerAddress,
			)
			if err != nil {
				return err
			}

			defer w.Stop()

			return w.Run(context.Background())
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "The worker id")
	cmd.Flags().StringVarP(&resourcesTopic, "resources-topic", "t", "", "The kafka topic where the resources are published.")

	return cmd
}

func CleanupWorkerCommand() *cobra.Command {
	var (
		id string
	)
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
			logger, err := zap.NewProduction()
			if err != nil {
				return err
			}

			cmd.SilenceUsage = true

			w, err := InitializeCleanupWorker(
				id,
				RabbitMQUsername,
				RabbitMQPassword,
				RabbitMQService,
				RabbitMQPort,
				DescribeCleanupJobsQueueName,
				ElasticSearchAddress,
				ElasticSearchUsername,
				ElasticSearchPassword,
				logger,
				PrometheusPushAddress,
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

func ConnectionWorkerCommand() *cobra.Command {
	var (
		id             string
		resourcesTopic string
	)
	cmd := &cobra.Command{
		PreRunE: func(cmd *cobra.Command, args []string) error {
			switch {
			case id == "":
				return errors.New("missing required flag 'id'")
			case resourcesTopic == "":
				return errors.New("missing required flag 'resources-topic'")
			default:
				return nil
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, err := zap.NewProduction()
			if err != nil {
				return err
			}

			cmd.SilenceUsage = true

			w, err := InitializeConnectionWorker(
				id,
				RabbitMQUsername,
				RabbitMQPassword,
				RabbitMQService,
				RabbitMQPort,
				DescribeConnectionJobsQueueName,
				DescribeConnectionResultsQueueName,
				strings.Split(KafkaService, ","),
				resourcesTopic,
				VaultAddress,
				VaultRoleName,
				VaultToken,
				VaultCaPath,
				VaultUseTLS,
				logger,
				ElasticSearchAddress,
				ElasticSearchUsername,
				ElasticSearchPassword,
				PrometheusPushAddress,
				RedisAddress,
				CacheAddress,
				JaegerAddress,
			)
			if err != nil {
				return err
			}

			defer w.Stop()

			return w.Run(context.Background())
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "The worker id")
	cmd.Flags().StringVarP(&resourcesTopic, "resources-topic", "t", "", "The kafka topic where the resources are published.")

	return cmd
}
