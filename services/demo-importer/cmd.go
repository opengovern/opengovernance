package demo_importer

import (
	"fmt"
	"github.com/kaytu-io/kaytu-util/pkg/config"
	es "github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/open-governance/services/demo-importer/fetch"
	"github.com/kaytu-io/open-governance/services/demo-importer/types"
	"github.com/kaytu-io/open-governance/services/demo-importer/worker"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func Command() *cobra.Command {
	var (
		cnf types.DemoImporterConfig
	)
	config.ReadFromEnv(&cnf, nil)
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}

	logger.Info("running", zap.String("es_address", cnf.ElasticSearch.Address), zap.String("es_arn", cnf.ElasticSearch.AssumeRoleArn))

	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {

			cmd.SilenceUsage = true

			_, err = fetch.GitClone(cnf, logger)
			if err != nil {
				return fmt.Errorf("failure while running git clone: %w", err)
			}

			esClient, err := es.NewClient(es.ClientConfig{
				Addresses:    []string{cnf.ElasticSearch.Address},
				Username:     &cnf.ElasticSearch.Username,
				Password:     &cnf.ElasticSearch.Password,
				IsOpenSearch: &cnf.ElasticSearch.IsOpenSearch,
				IsOnAks:      &cnf.ElasticSearch.IsOnAks,
				ExternalID:   &cnf.ElasticSearch.ExternalID,
			})
			if err != nil {
				return fmt.Errorf("failure while creating ES Client: %w", err)
			}

			err = worker.ImportJob(logger, esClient)
			if err != nil {
				return fmt.Errorf("failure while importing job: %w", err)
			}

			return nil
		},
	}

	return cmd
}
