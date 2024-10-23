package onboard

import (
	"context"
	"fmt"
	api3 "github.com/opengovern/og-util/pkg/api"
	"github.com/opengovern/og-util/pkg/httpclient"
	"github.com/opengovern/og-util/pkg/httpserver"
	"github.com/opengovern/og-util/pkg/koanf"
	"github.com/opengovern/og-util/pkg/vault"
	metadata "github.com/opengovern/opengovernance/pkg/metadata/client"
	"github.com/opengovern/opengovernance/pkg/onboard/config"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func Command() *cobra.Command {
	return &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			return start(cmd.Context())
		},
	}
}

func start(ctx context.Context) error {
	logger, err := zap.NewProduction()
	if err != nil {
		return fmt.Errorf("new logger: %w", err)
	}

	cfg := koanf.Provide("onboard", config.OnboardConfig{})

	mClient := metadata.NewMetadataServiceClient(cfg.Metadata.BaseURL)

	configured, err := mClient.VaultConfigured(&httpclient.Context{UserRole: api3.AdminRole})
	if err != nil {
		return err
	}
	if *configured != "True" {
		return fmt.Errorf("vault not configured")
	}

	var vaultSc vault.VaultSourceConfig
	switch cfg.Vault.Provider {
	case vault.AwsKMS:
		vaultSc, err = vault.NewKMSVaultSourceConfig(ctx, cfg.Vault.Aws, cfg.Vault.KeyId)
		if err != nil {
			logger.Error("failed to create vault source config", zap.Error(err))
			return err
		}
	case vault.AzureKeyVault:
		vaultSc, err = vault.NewAzureVaultClient(ctx, logger, cfg.Vault.Azure, cfg.Vault.KeyId)
		if err != nil {
			logger.Error("failed to create vault source config", zap.Error(err))
			return err
		}
	case vault.HashiCorpVault:
		vaultSc, err = vault.NewHashiCorpVaultClient(ctx, logger, cfg.Vault.HashiCorp, cfg.Vault.KeyId)
		if err != nil {
			logger.Error("failed to create vault source config", zap.Error(err))
			return err
		}
	}

	handler, err := InitializeHttpHandler(
		ctx,
		cfg.Postgres.Username, cfg.Postgres.Password, cfg.Postgres.Host, cfg.Postgres.Port, cfg.Postgres.DB, cfg.Postgres.SSLMode,
		cfg.Steampipe.Host, cfg.Steampipe.Port, cfg.Steampipe.DB, cfg.Steampipe.Username, cfg.Steampipe.Password,
		logger,
		vaultSc,
		cfg.Vault.KeyId,
		cfg.Inventory.BaseURL,
		cfg.Describe.BaseURL,
		cfg.Metadata.BaseURL,
		cfg.MasterAccessKey, cfg.MasterSecretKey,
	)
	if err != nil {
		return fmt.Errorf("init http handler: %w", err)
	}

	return httpserver.RegisterAndStart(ctx, logger, cfg.Http.Address, handler)
}
