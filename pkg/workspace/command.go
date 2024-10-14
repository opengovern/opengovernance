package workspace

import (
	"fmt"
	"github.com/opengovern/og-util/pkg/koanf"
	"github.com/opengovern/og-util/pkg/vault"
	"github.com/opengovern/opengovernance/pkg/workspace/config"
	vault2 "github.com/opengovern/opengovernance/pkg/workspace/vault"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := koanf.Provide("workspace", config.Config{})
			ctx := cmd.Context()

			logger, err := zap.NewProduction()
			if err != nil {
				return fmt.Errorf("new zap logger: %s", err)
			}

			if cfg.Vault.Provider == vault.HashiCorpVault {
				sealHandler, err := vault2.NewSealHandler(ctx, logger, cfg)
				if err != nil {
					return fmt.Errorf("new seal handler: %w", err)
				}
				// This blocks until vault is inited and unsealed
				sealHandler.Start(ctx)
			}

			s, err := NewServer(ctx, logger, cfg)
			if err != nil {
				return fmt.Errorf("new server: %w", err)
			}
			return s.Start(ctx)
		},
	}
	return cmd
}
