package assistant

import (
	complianceClient "github.com/kaytu-io/kaytu-engine/pkg/compliance/client"
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

			inventoryServiceClient := inventory.NewInventoryServiceClient(cnf.Inventory.BaseURL)
			complianceServiceClient := complianceClient.NewComplianceClient(cnf.Compliance.BaseURL)

			promptRepo := repository.NewPrompt(database)

			queryAssistant, err := openai.NewQueryAssistant(logger, cnf.OpenAI.Token, cnf.OpenAI.BaseURL, cnf.OpenAI.ModelName, complianceServiceClient, promptRepo)
			if err != nil {
				logger.Error("failed to create query assistant", zap.Error(err))
				return err
			}
			redirectionAssistant, err := openai.NewRedirectionAssistant(logger, cnf.OpenAI.Token, cnf.OpenAI.BaseURL, cnf.OpenAI.ModelName, promptRepo)
			if err != nil {
				logger.Error("failed to create redirection assistant", zap.Error(err))
				return err
			}

			queryAssistantActions, err := actions.NewQueryAssistantActions(queryAssistant, inventoryServiceClient, repository.NewRun(database))
			if err != nil {
				logger.Error("failed to create query assistant actions", zap.Error(err))
			}
			go queryAssistantActions.Run()

			cmd.SilenceUsage = true

			return httpserver.RegisterAndStart(
				logger,
				cnf.Http.Address,
				api.New(logger, queryAssistant, redirectionAssistant, database),
			)
		},
	}

	return cmd
}
