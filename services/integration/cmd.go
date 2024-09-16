package integration

import (
	"github.com/kaytu-io/kaytu-util/pkg/httpserver"
	"github.com/kaytu-io/kaytu-util/pkg/koanf"
	"github.com/kaytu-io/kaytu-util/pkg/vault"
	describe "github.com/kaytu-io/open-governance/pkg/describe/client"
	inventory "github.com/kaytu-io/open-governance/pkg/inventory/client"
	"github.com/kaytu-io/open-governance/services/integration/api"
	"github.com/kaytu-io/open-governance/services/integration/config"
	"github.com/kaytu-io/open-governance/services/integration/db"
	"github.com/kaytu-io/open-governance/services/integration/meta"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func Command() *cobra.Command {
	cnf := koanf.Provide("integration", config.IntegrationConfig{})

	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()

			logger, err := zap.NewProduction()
			if err != nil {
				return err
			}

			logger = logger.Named("integration")

			db, err := db.New(cnf.Postgres, logger)
			if err != nil {
				return err
			}

			var vaultSc vault.VaultSourceConfig
			switch cnf.Vault.Provider {
			case vault.AwsKMS:
				vaultSc, err = vault.NewKMSVaultSourceConfig(ctx, cnf.Vault.Aws, cnf.Vault.KeyId)
				if err != nil {
					logger.Error("failed to create vault source config", zap.Error(err))
					return err
				}
			case vault.AzureKeyVault:
				vaultSc, err = vault.NewAzureVaultClient(ctx, logger, cnf.Vault.Azure, cnf.Vault.KeyId)
				if err != nil {
					logger.Error("failed to create vault source config", zap.Error(err))
					return err
				}
			case vault.HashiCorpVault:
				vaultSc, err = vault.NewHashiCorpVaultClient(ctx, logger, cnf.Vault.HashiCorp, cnf.Vault.KeyId)
				if err != nil {
					logger.Error("failed to create vault source config", zap.Error(err))
					return err
				}
			}

			i := inventory.NewInventoryServiceClient(cnf.Inventory.BaseURL)
			d := describe.NewSchedulerServiceClient(cnf.Describe.BaseURL)
			m, err := meta.New(cnf.Metadata)
			if err != nil {
				return err
			}

			cmd.SilenceUsage = true

			return httpserver.RegisterAndStart(
				cmd.Context(),
				logger,
				cnf.Http.Address,
				api.New(logger, d, i, m, db, vaultSc, cnf.Vault.KeyId, cnf.MasterAccessKey, cnf.MasterSecretKey),
			)
		},
	}

	return cmd
}
