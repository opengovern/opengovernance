package insight

import (
	"errors"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const (
	InsightJobsQueueName    = "insight-jobs-queue"
	InsightResultsQueueName = "insight-results-queue"
)

var (
	ElasticSearchAddress  = os.Getenv("ES_ADDRESS")
	ElasticSearchUsername = os.Getenv("ES_USERNAME")
	ElasticSearchPassword = os.Getenv("ES_PASSWORD")

	RabbitMQService  = os.Getenv("RABBITMQ_SERVICE")
	RabbitMQPort     = 5672
	RabbitMQUsername = os.Getenv("RABBITMQ_USERNAME")
	RabbitMQPassword = os.Getenv("RABBITMQ_PASSWORD")

	KafkaService = os.Getenv("KAFKA_SERVICE")

	SteampipeHost     = os.Getenv("STEAMPIPE_HOST")
	SteampipePort     = os.Getenv("STEAMPIPE_PORT")
	SteampipeDb       = os.Getenv("STEAMPIPE_DB")
	SteampipeUser     = os.Getenv("STEAMPIPE_USERNAME")
	SteampipePassword = os.Getenv("STEAMPIPE_PASSWORD")

	PrometheusPushAddress = os.Getenv("PROMETHEUS_PUSH_ADDRESS")

	OnboardBaseURL = os.Getenv("ONBOARD_BASE_URL")

	S3Endpoint     = os.Getenv("S3_ENDPOINT")
	S3AccessKey    = os.Getenv("S3_ACCESS_KEY")
	S3AccessSecret = os.Getenv("S3_ACCESS_SECRET")
	S3Region       = os.Getenv("S3_REGION")
	S3Bucket       = os.Getenv("S3_BUCKET")

	CurrentWorkspaceID = os.Getenv("CURRENT_NAMESPACE")
)

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
				InsightJobsQueueName,
				InsightResultsQueueName,
				strings.Split(KafkaService, ","),
				resourcesTopic,
				logger,
				PrometheusPushAddress,
				SteampipeHost,
				SteampipePort,
				SteampipeDb,
				SteampipeUser,
				SteampipePassword,
				ElasticSearchAddress,
				ElasticSearchUsername,
				ElasticSearchPassword,
				OnboardBaseURL,
				S3Endpoint, S3AccessKey,
				S3AccessSecret, S3Region,
				S3Bucket,
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
