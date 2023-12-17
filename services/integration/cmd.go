package integration

import (
	"context"

	describe "github.com/kaytu-io/kaytu-engine/pkg/describe/client"
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	inventory "github.com/kaytu-io/kaytu-engine/pkg/inventory/client"
	"github.com/kaytu-io/kaytu-engine/services/integration/api"
	"github.com/kaytu-io/kaytu-engine/services/integration/config"
	"github.com/kaytu-io/kaytu-engine/services/integration/db"
	"github.com/kaytu-io/kaytu-engine/services/integration/meta"
	"github.com/kaytu-io/kaytu-engine/services/integration/steampipe"
	"github.com/kaytu-io/kaytu-util/pkg/koanf"
	"github.com/kaytu-io/kaytu-util/pkg/vault"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func Command() *cobra.Command {
	cnf := koanf.Provide("", config.OnboardConfig{})

	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, _ []string) error {
			logger, err := zap.NewProduction()
			if err != nil {
				return err
			}

			logger = logger.Named("onboard")

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

			api.New(logger, d, i, m, s, db, kms, cnf.KMS.ARN, cnf.MasterAccessKey, cnf.MasterSecretKey)

			cmd.SilenceUsage = true

			return httpserver.RegisterAndStart(logger, cnf.Http.Address, nil)
		},
	}

	return cmd
}
