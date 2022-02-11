package steampipe

import (
	"errors"
	"github.com/spf13/cobra"
	"os"
)

const (
	JobsQueueName    = "steampipe-jobs-queue"
	ResultsQueueName = "steampipe-results-queue"
)

type ElasticSearchConfig struct {
	Address  string
	Username string
	Password string
}

type RabbitMQConfig struct {
	Host     string
	Port     int
	Username string
	Password string
}

type S3ClientConfig struct {
	Key      string
	Secret   string
	Endpoint string
	Region   string
	Bucket   string
}

type Config struct {
	S3Client      S3ClientConfig
	RabbitMQ      RabbitMQConfig
	ElasticSearch ElasticSearchConfig
}

func WorkerCommand() *cobra.Command {
	var (
		id     string
		config Config
	)

	config.RabbitMQ.Host = os.Getenv("RABBITMQ_SERVICE")
	config.RabbitMQ.Port = 5672
	config.RabbitMQ.Username = os.Getenv("RABBITMQ_USERNAME")
	config.RabbitMQ.Password = os.Getenv("RABBITMQ_PASSWORD")

	config.S3Client.Endpoint = os.Getenv("S3_ENDPOINT")
	config.S3Client.Key = os.Getenv("S3_KEY")
	config.S3Client.Secret = os.Getenv("S3_SECRET")
	config.S3Client.Region = os.Getenv("S3_REGION")
	config.S3Client.Bucket = os.Getenv("S3_BUCKET")

	config.ElasticSearch.Address = os.Getenv("ES_ADDRESS")
	config.ElasticSearch.Username = os.Getenv("ES_USERNAME")
	config.ElasticSearch.Password = os.Getenv("ES_PASSWORD")

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

			w, err := InitializeWorker(
				id,
				config,
				JobsQueueName,
				ResultsQueueName,
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
