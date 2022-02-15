package compliance_report

import (
	"errors"
	"github.com/spf13/cobra"
	"os"
)

const (
	JobsQueueName    = "compliance-report-jobs-queue"
	ResultsQueueName = "compliance-report-results-queue"
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
	Bucket   string
}

type VaultConfig struct {
	Address  string
	RoleName string
	Token    string
}

type Config struct {
	S3Client      S3ClientConfig
	RabbitMQ      RabbitMQConfig
	ElasticSearch ElasticSearchConfig
	Vault         VaultConfig
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
	config.S3Client.Bucket = os.Getenv("S3_BUCKET")

	config.ElasticSearch.Address = os.Getenv("ES_ADDRESS")
	config.ElasticSearch.Username = os.Getenv("ES_USERNAME")
	config.ElasticSearch.Password = os.Getenv("ES_PASSWORD")

	config.Vault.Address = os.Getenv("VAULT_ADDRESS")
	config.Vault.Token = os.Getenv("VAULT_TOKEN")
	config.Vault.RoleName = os.Getenv("VAULT_ROLE")

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
