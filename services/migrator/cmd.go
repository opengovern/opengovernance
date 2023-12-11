package migrator

import (
	"fmt"
	config2 "github.com/kaytu-io/kaytu-engine/services/migrator/config"
	"github.com/kaytu-io/kaytu-engine/services/migrator/job"
	"github.com/kaytu-io/kaytu-util/pkg/config"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func Command() *cobra.Command {
	var (
		cnf config2.MigratorConfig
	)
	config.ReadFromEnv(&cnf, nil)

	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, err := zap.NewProduction()
			if err != nil {
				return err
			}

			cmd.SilenceUsage = true

			j, err := job.InitializeJob(cnf, logger)
			if err != nil {
				return fmt.Errorf("failed to initialize job: %w", err)
			}

			return j.Run()
		},
	}

	return cmd
}
