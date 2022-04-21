package compliance_report

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
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
	CaPath   string
}

type KafkaConfig struct {
	Addresses string
	Topic     string
}

type Config struct {
	RabbitMQ      RabbitMQConfig
	ElasticSearch ElasticSearchConfig
	Kafka         KafkaConfig
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

	config.ElasticSearch.Address = os.Getenv("ES_ADDRESS")
	config.ElasticSearch.Username = os.Getenv("ES_USERNAME")
	config.ElasticSearch.Password = os.Getenv("ES_PASSWORD")

	config.Kafka.Addresses = os.Getenv("KAFKA_ADDRESSES")
	config.Kafka.Topic = os.Getenv("KAFKA_TOPIC")

	config.Vault.Address = os.Getenv("VAULT_ADDRESS")
	config.Vault.Token = os.Getenv("VAULT_TOKEN")
	config.Vault.RoleName = os.Getenv("VAULT_ROLE")
	config.Vault.CaPath = os.Getenv("VAULT_TLS_CA_PATH")

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
				config,
				JobsQueueName,
				ResultsQueueName,
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
