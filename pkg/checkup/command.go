package checkup

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

const (
	CheckupJobsQueueName    = "checkup-jobs-queue"
	CheckupResultsQueueName = "checkup-results-queue"
)

var (
	RabbitMQService  = os.Getenv("RABBITMQ_SERVICE")
	RabbitMQPort     = 5672
	RabbitMQUsername = os.Getenv("RABBITMQ_USERNAME")
	RabbitMQPassword = os.Getenv("RABBITMQ_PASSWORD")

	PrometheusPushAddress = os.Getenv("PROMETHEUS_PUSH_ADDRESS")

	OnboardBaseURL = os.Getenv("ONBOARD_BASE_URL")
)

func WorkerCommand() *cobra.Command {
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

			w, err := InitializeWorker(
				id,
				RabbitMQUsername,
				RabbitMQPassword,
				RabbitMQService,
				RabbitMQPort,
				CheckupJobsQueueName,
				CheckupResultsQueueName,
				logger,
				PrometheusPushAddress,
				OnboardBaseURL,
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
