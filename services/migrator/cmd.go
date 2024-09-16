package migrator

import (
	"fmt"
	"github.com/kaytu-io/kaytu-util/pkg/config"
	config2 "github.com/kaytu-io/open-governance/services/migrator/config"
	"github.com/kaytu-io/open-governance/services/migrator/job"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func Command() *cobra.Command {
	var (
		cnf config2.MigratorConfig
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

			j, err := job.InitializeJob(cnf, logger)
			if err != nil {
				return fmt.Errorf("failed to initialize job: %w", err)
			}

			return j.Run(cmd.Context())
		},
	}

	return cmd
}
