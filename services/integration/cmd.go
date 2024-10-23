package integration

import (
	"fmt"
	api3 "github.com/opengovern/og-util/pkg/api"
	"github.com/opengovern/og-util/pkg/httpclient"
	"github.com/opengovern/og-util/pkg/httpserver"
	"github.com/opengovern/og-util/pkg/koanf"
	"github.com/opengovern/og-util/pkg/vault"
	describe "github.com/opengovern/opengovernance/pkg/describe/client"
	inventory "github.com/opengovern/opengovernance/pkg/inventory/client"
	metadata "github.com/opengovern/opengovernance/pkg/metadata/client"
	"github.com/opengovern/opengovernance/services/integration/api"
	"github.com/opengovern/opengovernance/services/integration/config"
	"github.com/opengovern/opengovernance/services/integration/db"
	"github.com/opengovern/opengovernance/services/integration/meta"
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

			i := inventory.NewInventoryServiceClient(cnf.Inventory.BaseURL)
			d := describe.NewSchedulerServiceClient(cnf.Describe.BaseURL)
			mClient := metadata.NewMetadataServiceClient(cnf.Metadata.BaseURL)

			configured, err := mClient.VaultConfigured(&httpclient.Context{UserRole: api3.AdminRole})
			if err != nil {
				return err
			}
			if *configured != "True" {
				return fmt.Errorf("vault not configured")
			}

			m, err := meta.New(cnf.Metadata)
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

			cmd.SilenceUsage = true

			return httpserver.RegisterAndStart(
				cmd.Context(),
				logger,
				cnf.Http.Address,
				api.New(logger, d, i, mClient, m, db, vaultSc, cnf.Vault.KeyId, cnf.MasterAccessKey, cnf.MasterSecretKey),
			)
		},
	}

	return cmd
}
