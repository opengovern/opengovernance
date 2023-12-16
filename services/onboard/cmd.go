package onboard

import (
	"context"

	describe "github.com/kaytu-io/kaytu-engine/pkg/describe/client"
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	inventory "github.com/kaytu-io/kaytu-engine/pkg/inventory/client"
	"github.com/kaytu-io/kaytu-engine/services/onboard/api"
	"github.com/kaytu-io/kaytu-engine/services/onboard/config"
	"github.com/kaytu-io/kaytu-engine/services/onboard/db"
	"github.com/kaytu-io/kaytu-engine/services/onboard/meta"
	"github.com/kaytu-io/kaytu-engine/services/onboard/steampipe"
	"github.com/kaytu-io/kaytu-util/pkg/koanf"
	"github.com/kaytu-io/kaytu-util/pkg/queue"
	"github.com/kaytu-io/kaytu-util/pkg/vault"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func Command() *cobra.Command {
	cnf := koanf.Provide("onboard", config.OnboardConfig{})

	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, _ []string) error {
			logger, err := zap.NewProduction()
			if err != nil {
				return err
			}

			logger = logger.Named("onboard")

			// setup source events queue
			var qCfg queue.Config
			qCfg.Server.Username = cnf.RabbitMQ.Username
			qCfg.Server.Password = cnf.RabbitMQ.Password
			qCfg.Server.Host = cnf.RabbitMQ.Service
			qCfg.Server.Port = 5672
			qCfg.Queue.Name = config.SourceEventsQueueName
			qCfg.Queue.Durable = true
			qCfg.Producer.ID = "onboard-service"
			q, err := queue.New(qCfg)
			if err != nil {
				return err
			}

			db.New(cnf.Postgres, logger)
			steampipe.New(cnf.Steampipe, logger)

			// TODO (parham) why access-key and secret-key are empty?
			vault.NewKMSVaultSourceConfig(context.Background(), "", "", cnf.KMS.Region)

			inventory.NewInventoryServiceClient(cnf.Inventory.BaseURL)
			describe.NewSchedulerServiceClient(cnf.Describe.BaseURL)
			meta.New(cnf.Metadata)
			api.New(logger, q)

			cmd.SilenceUsage = true

			return httpserver.RegisterAndStart(logger, cnf.Http.Address, nil)
		},
	}

	return cmd
}
