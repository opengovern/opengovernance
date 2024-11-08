package checkup

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var (
	PrometheusPushAddress = os.Getenv("PROMETHEUS_PUSH_ADDRESS")
	IntegrationBaseUrl    = os.Getenv("INTEGRATION_BASE_URL")
	NATSAddress           = os.Getenv("NATS_URL")
)

func WorkerCommand() *cobra.Command {
	var id string
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

			w, err := NewWorker(
				id,
				NATSAddress,
				logger,
				PrometheusPushAddress,
				IntegrationBaseUrl,
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
