package subscription

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	"github.com/kaytu-io/kaytu-engine/services/subscription/api"
	config2 "github.com/kaytu-io/kaytu-engine/services/subscription/config"
	"github.com/kaytu-io/kaytu-engine/services/subscription/db"
	"github.com/kaytu-io/kaytu-engine/services/subscription/metering"
	"github.com/kaytu-io/kaytu-util/pkg/config"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func Command() *cobra.Command {
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

			pdb, err := db.NewDatabase(cnf.Postgres, logger)
			if err != nil {
				return fmt.Errorf("new postgres client: %w", err)
			}

			handler, err := api.InitializeHttpServer(
				logger,
				cnf,
				pdb,
			)
			if err != nil {
				return err
			}

			meteringService, err := metering.New(logger, cnf, pdb)
			if err != nil {
				return err
			}

			go meteringService.Run()

			return httpserver.RegisterAndStart(logger, cnf.Http.Address, handler)
		},
	}

	return cmd
}
