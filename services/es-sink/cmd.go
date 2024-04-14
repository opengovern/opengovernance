package es_sink

import (
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	"github.com/kaytu-io/kaytu-engine/pkg/jq"
	"github.com/kaytu-io/kaytu-engine/services/es-sink/api"
	"github.com/kaytu-io/kaytu-engine/services/es-sink/config"
	"github.com/kaytu-io/kaytu-engine/services/es-sink/service"
	es "github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/koanf"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func Command() *cobra.Command {
	cnf := koanf.Provide("essink", config.EsSinkConfig{})

	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			logger, err := zap.NewProduction()
			if err != nil {
				return err
			}

			logger = logger.Named("es-sink")

			cmd.SilenceUsage = true

			esClient, err := es.NewClient(es.ClientConfig{
				Addresses:     []string{cnf.ElasticSearch.Address},
				Username:      &cnf.ElasticSearch.Username,
				Password:      &cnf.ElasticSearch.Password,
				IsOpenSearch:  &cnf.ElasticSearch.IsOpenSearch,
				IsOnAks:       &cnf.ElasticSearch.IsOnAks,
				AwsRegion:     &cnf.ElasticSearch.AWSRegion,
				AssumeRoleArn: &cnf.ElasticSearch.AssumeRoleARN,
				ExternalID:    &cnf.ElasticSearch.ExternalID,
			})

			if err != nil {
				logger.Error("failed to create es client", zap.Error(err))
				return err
			}

			nats, err := jq.New(cnf.NATS.URL, logger)
			if err != nil {
				logger.Error("failed to create nats client", zap.Error(err))
				return err
			}

			sinkService, err := service.NewEsSinkService(ctx, logger, esClient, nats)
			if err != nil {
				logger.Error("failed to create es sink service", zap.Error(err))
				return err
			}

			go sinkService.Start(ctx)

			return httpserver.RegisterAndStart(
				ctx,
				logger,
				cnf.Http.Address,
				api.New(logger, sinkService),
			)
		},
	}

	return cmd
}