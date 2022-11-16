package summarizer

import (
	"errors"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const (
	SummarizerJobsQueueName    = "summarizer-jobs-queue"
	SummarizerResultsQueueName = "summarizer-results-queue"
)

var (
	PostgreSQLHost     = os.Getenv("POSTGRESQL_HOST")
	PostgreSQLPort     = os.Getenv("POSTGRESQL_PORT")
	PostgreSQLDb       = os.Getenv("POSTGRESQL_DB")
	PostgreSQLUser     = os.Getenv("POSTGRESQL_USERNAME")
	PostgreSQLPassword = os.Getenv("POSTGRESQL_PASSWORD")
	PostgreSQLSSLMode  = os.Getenv("POSTGRESQL_SSLMODE")

	ElasticSearchAddress  = os.Getenv("ES_ADDRESS")
	ElasticSearchUsername = os.Getenv("ES_USERNAME")
	ElasticSearchPassword = os.Getenv("ES_PASSWORD")

	RabbitMQService  = os.Getenv("RABBITMQ_SERVICE")
	RabbitMQPort     = 5672
	RabbitMQUsername = os.Getenv("RABBITMQ_USERNAME")
	RabbitMQPassword = os.Getenv("RABBITMQ_PASSWORD")

	KafkaService = os.Getenv("KAFKA_SERVICE")

	PrometheusPushAddress = os.Getenv("PROMETHEUS_PUSH_ADDRESS")
)

func WorkerCommand() *cobra.Command {
	var (
		id         string
		kafkaTopic string
	)
	cmd := &cobra.Command{
		PreRunE: func(cmd *cobra.Command, args []string) error {
			switch {
			case id == "":
				return errors.New("missing required flag 'id'")
			case kafkaTopic == "":
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

			w, err := InitializeWorker(
				id,
				RabbitMQUsername,
				RabbitMQPassword,
				RabbitMQService,
				RabbitMQPort,
				SummarizerJobsQueueName,
				SummarizerResultsQueueName,
				strings.Split(KafkaService, ","),
				kafkaTopic,
				logger,
				PrometheusPushAddress,
				ElasticSearchAddress,
				ElasticSearchUsername,
				ElasticSearchPassword,
				PostgreSQLHost,
				PostgreSQLPort,
				PostgreSQLDb,
				PostgreSQLUser,
				PostgreSQLPassword,
				PostgreSQLSSLMode,
			)
			if err != nil {
				return err
			}

			defer w.Stop()

			return w.Run()
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "The worker id")
	cmd.Flags().StringVarP(&kafkaTopic, "resources-topic", "t", "", "The kafka topic where the resources are published.")

	return cmd
}
