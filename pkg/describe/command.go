package describe

import (
	"errors"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

const (
	DescribeJobsQueueName            = "describe-jobs-queue"
	DescribeResultsQueueName         = "describe-results-queue"
	DescribeCleanupJobsQueueName     = "describe-cleanup-jobs-queue"
	ComplianceReportJobsQueueName    = "compliance-report-jobs-queue"
	ComplianceReportResultsQueueName = "compliance-report-results-queue"
	SourceEventsQueueName            = "source-events-queue"
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

	VaultAddress  = os.Getenv("VAULT_ADDRESS")
	VaultToken    = os.Getenv("VAULT_TOKEN")
	VaultRoleName = os.Getenv("VAULT_ROLE")

	ElasticSearchAddress  = os.Getenv("ES_ADDRESS")
	ElasticSearchUsername = os.Getenv("ES_USERNAME")
	ElasticSearchPassword = os.Getenv("ES_PASSWORD")

	HttpServerAddress = os.Getenv("HTTP_ADDRESS")
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
				DescribeCleanupJobsQueueName,
				ComplianceReportJobsQueueName,
				ComplianceReportResultsQueueName,
				SourceEventsQueueName,
				PostgreSQLUser,
				PostgreSQLPassword,
				PostgreSQLHost,
				PostgreSQLPort,
				PostgreSQLDb,
				HttpServerAddress,
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
			)
			if err != nil {
				return err
			}

			defer w.Stop()

			return w.Run()
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
