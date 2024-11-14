package metadata

import (
	"context"
	"fmt"
	"strings"

	dexApi "github.com/dexidp/dex/api/v2"
	"github.com/opengovern/og-util/pkg/httpserver"
	"github.com/opengovern/og-util/pkg/koanf"
	"github.com/opengovern/og-util/pkg/vault"
	"github.com/opengovern/opengovernance/services/metadata/config"
	vault2 "github.com/opengovern/opengovernance/services/metadata/vault"
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

	dexClient, err := newDexClient(cfg.DexGrpcAddr)
	if err != nil {
		logger.Error("Auth Migrator: failed to create dex client", zap.Error(err))
		return err
	}

	publicUris := strings.Split(cfg.DexPublicClientRedirectUris, ",")

	publicClientResp, _ := dexClient.GetClient(ctx, &dexApi.GetClientReq{
		Id: "public-client",
	})

	logger.Info("public URIS", zap.Any("uris", publicUris))

	if publicClientResp != nil && publicClientResp.Client != nil {
		publicClientReq := dexApi.UpdateClientReq{
			Id:           "public-client",
			Name:         "Public Client",
			RedirectUris: publicUris,
		}

		_, err = dexClient.UpdateClient(ctx, &publicClientReq)
		if err != nil {
			logger.Error("Auth Migrator: failed to create dex public client", zap.Error(err))
			return err
		}
	} else {
		publicClientReq := dexApi.CreateClientReq{
			Client: &dexApi.Client{
				Id:           "public-client",
				Name:         "Public Client",
				RedirectUris: publicUris,
				Public:       true,
			},
		}

		_, err = dexClient.CreateClient(ctx, &publicClientReq)
		if err != nil {
			logger.Error("Auth Migrator: failed to create dex public client", zap.Error(err))
			return err
		}
	}

	privateUris := strings.Split(cfg.DexPrivateClientRedirectUris, ",")

	logger.Info("private URIS", zap.Any("uris", privateUris))

	privateClientResp, _ := dexClient.GetClient(ctx, &dexApi.GetClientReq{
		Id: "private-client",
	})
	if privateClientResp != nil && privateClientResp.Client != nil {
		privateClientReq := dexApi.UpdateClientReq{
			Id:           "private-client",
			Name:         "Private Client",
			RedirectUris: privateUris,
		}

		_, err = dexClient.UpdateClient(ctx, &privateClientReq)
		if err != nil {
			logger.Error("Auth Migrator: failed to create dex private client", zap.Error(err))
			return err
		}
	} else {
		privateClientReq := dexApi.CreateClientReq{
			Client: &dexApi.Client{
				Id:           "private-client",
				Name:         "Private Client",
				RedirectUris: privateUris,
				Secret:       "secret",
			},
		}

		_, err = dexClient.CreateClient(ctx, &privateClientReq)
		if err != nil {
			logger.Error("Auth Migrator: failed to create dex private client", zap.Error(err))
			return err
		}
	}

	handler, err := InitializeHttpHandler(
		cfg,
		logger,
		dexClient,
	)
	if err != nil {
		return fmt.Errorf("init http handler: %w", err)
	}

	return httpserver.RegisterAndStart(ctx, logger, cfg.Http.Address, handler)
}
