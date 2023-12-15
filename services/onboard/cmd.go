package onboard

import (
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	config2 "github.com/kaytu-io/kaytu-engine/services/onboard/config"
	"github.com/kaytu-io/kaytu-util/pkg/config"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func Command() *cobra.Command {
	var cnf config2.OnboardConfig
	config.ReadFromEnv(&cnf, nil)

	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, _ []string) error {
			logger, err := zap.NewProduction()
			if err != nil {
				return err
			}

			cmd.SilenceUsage = true

			return httpserver.RegisterAndStart(logger, cnf.Http.Address, nil)
		},
	}

	return cmd
}
