package rego

import (
	config2 "github.com/opengovern/og-util/pkg/config"
	"github.com/opengovern/og-util/pkg/httpserver"
	"github.com/opengovern/og-util/pkg/steampipe"
	integration_type "github.com/opengovern/opencomply/services/integration/integration-type"
	"github.com/opengovern/opencomply/services/rego/api"
	"github.com/opengovern/opencomply/services/rego/config"
	"github.com/opengovern/opencomply/services/rego/service"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"time"
)

func Command() *cobra.Command {
	var cnf config.RegoConfig
	config2.ReadFromEnv(&cnf, nil)

	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			logger, err := zap.NewProduction()
			if err != nil {
				return err
			}

			logger = logger.Named("rego")

			for _, integrationType := range integration_type.IntegrationTypes {
				describerConfig := integrationType.GetConfiguration()
				err := steampipe.PopulateSteampipeConfig(cnf.ElasticSearch, describerConfig.SteampipePluginName)
				if err != nil {
					return err
				}
			}
			if err := steampipe.PopulateOpenGovernancePluginSteampipeConfig(cnf.ElasticSearch, cnf.Steampipe); err != nil {
				return err
			}

			time.Sleep(2 * time.Minute)

			steampipeConn, err := steampipe.StartSteampipeServiceAndGetConnection(logger)
			if err != nil {
				return err
			}

			regoEngine, err := service.NewRegoEngine(ctx, logger, steampipeConn)
			if err != nil {
				return err
			}

			return httpserver.RegisterAndStart(
				ctx,
				logger,
				cnf.Http.Address,
				api.New(logger, regoEngine),
			)
		},
	}

	return cmd
}
