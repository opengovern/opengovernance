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

			db, err := db.New(cnf.Postgres, logger)
			if err != nil {
				return err
			}

			s, err := steampipe.New(cnf.Steampipe, logger)
			if err != nil {
				return err
			}

			// TODO (parham) why access-key and secret-key are empty?
			kms, err := vault.NewKMSVaultSourceConfig(context.Background(), "", "", cnf.KMS.Region)
			if err != nil {
				return err
			}

			i := inventory.NewInventoryServiceClient(cnf.Inventory.BaseURL)
			d := describe.NewSchedulerServiceClient(cnf.Describe.BaseURL)
			m, err := meta.New(cnf.Metadata)
			if err != nil {
				return err
			}

			api.New(logger, q, d, i, m, s, db, kms)

			cmd.SilenceUsage = true

			return httpserver.RegisterAndStart(logger, cnf.Http.Address, nil)
		},
	}

	return cmd
}
