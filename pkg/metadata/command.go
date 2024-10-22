package metadata

import (
	"context"
	"fmt"
	"github.com/opengovern/og-util/pkg/httpserver"
	"github.com/opengovern/og-util/pkg/koanf"
	"github.com/opengovern/opengovernance/pkg/metadata/config"
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

	handler, err := InitializeHttpHandler(
		cfg,
		logger,
	)
	if err != nil {
		return fmt.Errorf("init http handler: %w", err)
	}

	return httpserver.RegisterAndStart(ctx, logger, cfg.Http.Address, handler)
}
