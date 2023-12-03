package subscription

import (
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	"github.com/kaytu-io/kaytu-engine/services/subscription/api"
	config2 "github.com/kaytu-io/kaytu-engine/services/subscription/config"
	"github.com/kaytu-io/kaytu-util/pkg/config"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func SubscriptionServiceCommand() *cobra.Command {
	var (
		cnf config2.SubscriptionConfig
	)
	config.ReadFromEnv(&cnf, nil)

	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, err := zap.NewProduction()
			if err != nil {
				return err
			}

			cmd.SilenceUsage = true

			handler, err := api.InitializeHttpServer(
				logger,
				cnf,
			)
			if err != nil {
				return err
			}

			return httpserver.RegisterAndStart(logger, cnf.Http.Address, handler)
		},
	}

	return cmd
}
