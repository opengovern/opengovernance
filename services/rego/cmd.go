package rego

import (
	config2 "github.com/opengovern/og-util/pkg/config"
	"github.com/opengovern/og-util/pkg/httpserver"
	"github.com/opengovern/og-util/pkg/koanf"
	"github.com/opengovern/og-util/pkg/steampipe"
	integration_type "github.com/opengovern/opencomply/services/integration/integration-type"
	"github.com/opengovern/opencomply/services/rego/api"
	"github.com/opengovern/opencomply/services/rego/config"
	"github.com/opengovern/opencomply/services/rego/internal"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"time"
)

func Command() *cobra.Command {
	cnf := koanf.Provide("rego", config.RegoConfig{})

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
				err := steampipe.PopulateSteampipeConfig(config2.ElasticSearch{
					Address:           cnf.ElasticSearch.Address,
					Username:          cnf.ElasticSearch.Username,
					Password:          cnf.ElasticSearch.Password,
					IsOpenSearch:      cnf.ElasticSearch.IsOpenSearch,
					IsOnAks:           cnf.ElasticSearch.IsOnAks,
					AwsRegion:         cnf.ElasticSearch.AWSRegion,
					AssumeRoleArn:     cnf.ElasticSearch.AssumeRoleARN,
					ExternalID:        cnf.ElasticSearch.ExternalID,
					IngestionEndpoint: cnf.ElasticSearch.IngestionEndpoint,
				}, describerConfig.SteampipePluginName)
				if err != nil {
					return err
				}
			}
			if err := steampipe.PopulateOpenGovernancePluginSteampipeConfig(config2.ElasticSearch{
				Address:           cnf.ElasticSearch.Address,
				Username:          cnf.ElasticSearch.Username,
				Password:          cnf.ElasticSearch.Password,
				IsOpenSearch:      cnf.ElasticSearch.IsOpenSearch,
				IsOnAks:           cnf.ElasticSearch.IsOnAks,
				AwsRegion:         cnf.ElasticSearch.AWSRegion,
				AssumeRoleArn:     cnf.ElasticSearch.AssumeRoleARN,
				ExternalID:        cnf.ElasticSearch.ExternalID,
				IngestionEndpoint: cnf.ElasticSearch.IngestionEndpoint,
			}, config2.Postgres{
				Host:            cnf.Steampipe.Host,
				Port:            cnf.Steampipe.Port,
				DB:              cnf.Steampipe.DB,
				Username:        cnf.Steampipe.Username,
				Password:        cnf.Steampipe.Password,
				SSLMode:         cnf.Steampipe.SSLMode,
				MaxIdleConns:    cnf.Steampipe.MaxIdleConns,
				MaxOpenConns:    cnf.Steampipe.MaxOpenConns,
				ConnMaxIdleTime: cnf.Steampipe.ConnMaxIdleTime,
				ConnMaxLifetime: cnf.Steampipe.ConnMaxLifetime,
			}); err != nil {
				return err
			}

			time.Sleep(2 * time.Minute)

			steampipeConn, err := steampipe.StartSteampipeServiceAndGetConnection(logger)
			if err != nil {
				return err
			}

			regoEngine, err := internal.NewRegoEngine(ctx, logger, steampipeConn)
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
