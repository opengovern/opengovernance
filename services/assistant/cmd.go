package assistant

import (
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	inventory "github.com/kaytu-io/kaytu-engine/pkg/inventory/client"
	"github.com/kaytu-io/kaytu-engine/services/assistant/actions"
	"github.com/kaytu-io/kaytu-engine/services/assistant/api"
	"github.com/kaytu-io/kaytu-engine/services/assistant/config"
	"github.com/kaytu-io/kaytu-engine/services/assistant/db"
	"github.com/kaytu-io/kaytu-engine/services/assistant/openai"
	"github.com/kaytu-io/kaytu-engine/services/assistant/repository"
	"github.com/kaytu-io/kaytu-util/pkg/koanf"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func Command() *cobra.Command {
	cnf := koanf.Provide("assistant", config.AssistantConfig{})

	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, _ []string) error {
			logger, err := zap.NewProduction()
			if err != nil {
				return err
			}

			logger = logger.Named("assistant")

			database, err := db.New(cnf.Postgres, logger)
			if err != nil {
				return err
			}

			i := inventory.NewInventoryServiceClient(cnf.Inventory.BaseURL)
			oc, err := openai.New(cnf.OpenAI.Token, cnf.OpenAI.BaseURL, cnf.OpenAI.ModelName, i)
			if err != nil {
				return err
			}

			a := actions.New(oc, i, repository.NewRun(database))
			go a.Run()

			cmd.SilenceUsage = true

			return httpserver.RegisterAndStart(
				logger,
				cnf.Http.Address,
				api.New(logger, oc, database),
			)
		},
	}

	return cmd
}
