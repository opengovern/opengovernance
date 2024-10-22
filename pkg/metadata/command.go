package metadata

import (
	"context"
	"fmt"
	"github.com/opengovern/og-util/pkg/httpserver"
	"github.com/opengovern/og-util/pkg/koanf"
	"github.com/opengovern/og-util/pkg/vault"
	"github.com/opengovern/opengovernance/pkg/metadata/config"
	vault2 "github.com/opengovern/opengovernance/pkg/metadata/vault"
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
	cfg := koanf.Provide("metadata", config.Config{})

	logger, err := zap.NewProduction()
	if err != nil {
		return fmt.Errorf("new logger: %w", err)
	}

	if cfg.Vault.Provider == vault.HashiCorpVault {
		sealHandler, err := vault2.NewSealHandler(ctx, logger, cfg)
		if err != nil {
			return fmt.Errorf("new seal handler: %w", err)
		}
		// This blocks until vault is inited and unsealed
		sealHandler.Start(ctx)
	}

	handler, err := InitializeHttpHandler(
		cfg,
		logger,
	)
	if err != nil {
		return fmt.Errorf("init http handler: %w", err)
	}

	return httpserver.RegisterAndStart(ctx, logger, cfg.Http.Address, handler)
}
