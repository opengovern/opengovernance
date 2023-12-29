package subscription

import (
	client2 "github.com/kaytu-io/kaytu-engine/pkg/auth/client"
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	workspaceClient "github.com/kaytu-io/kaytu-engine/pkg/workspace/client"
	"github.com/kaytu-io/kaytu-engine/services/subscription/api"
	config2 "github.com/kaytu-io/kaytu-engine/services/subscription/config"
	"github.com/kaytu-io/kaytu-engine/services/subscription/db"
	"github.com/kaytu-io/kaytu-engine/services/subscription/jobs"
	"github.com/kaytu-io/kaytu-engine/services/subscription/service"
	"github.com/kaytu-io/kaytu-util/pkg/koanf"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func Command() *cobra.Command {
	cnf := koanf.Provide("subscription", config2.SubscriptionConfig{})

	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			logger, err := zap.NewProduction()
			if err != nil {
				return err
			}
			logger = logger.Named("subscription")

			pdb, err := db.NewDatabase(cnf.Postgres, logger)
			if err != nil {
				return err
			}

			w := workspaceClient.NewWorkspaceClient(cnf.Workspace.BaseURL)
			a := client2.NewAuthServiceClient(cnf.Auth.BaseURL)

			meteringService := service.NewMeteringService(logger, pdb, cnf, w, a)

			go jobs.GenerateMeters(meteringService, logger)
			return httpserver.RegisterAndStart(
				logger,
				cnf.Http.Address,
				api.New(logger, pdb, w, meteringService),
			)
		},
	}

	return cmd
}
