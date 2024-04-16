package wastage

import (
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	"github.com/kaytu-io/kaytu-engine/services/wastage/api"
	"github.com/kaytu-io/kaytu-engine/services/wastage/config"
	"github.com/kaytu-io/kaytu-engine/services/wastage/cost"
	"github.com/kaytu-io/kaytu-util/pkg/koanf"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func Command() *cobra.Command {
	cnf := koanf.Provide("wastage", config.WastageConfig{})

	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, _ []string) error {
			logger, err := zap.NewProduction()
			if err != nil {
				return err
			}

			logger = logger.Named("wastage")

			cmd.SilenceUsage = true

			if cnf.Http.Address == "" {
				cnf.Http.Address = "localhost:8000"
				cnf.Pennywise.BaseURL = "http://localhost:8080"
			}
			costSvc := cost.New(cnf.Pennywise.BaseURL)
			return httpserver.RegisterAndStart(
				cmd.Context(),
				logger,
				cnf.Http.Address,
				api.New(costSvc, logger),
			)
		},
	}

	return cmd
}
