package vault

import (
	"context"
	"crypto/rand"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/config"
	"github.com/kaytu-io/kaytu-util/pkg/vault"
	"go.uber.org/zap"
)

func GetNewWorkspaceVaultKeyId(ctx context.Context, logger *zap.Logger, handler *vault.AzureVaultSecretHandler, cfg config.Config, workspaceId string) (string, error) {
	switch cfg.Vault.Provider {
	case vault.AwsKMS:
		return cfg.Vault.KeyId, nil
	case vault.AzureKeyVault:
		// create new aes key
		b := make([]byte, 32)
		_, err := rand.Read(b)
		if err != nil {
			logger.Error("failed to generate random bytes", zap.Error(err))
			return "", err
		}

		id, err := handler.SetSecret(ctx, fmt.Sprintf("client-creds-key-%s", workspaceId), b)
		if err != nil {
			logger.Error("failed to set secret", zap.Error(err))
			return "", err
		}
		return id, nil
	default:
		logger.Error("unsupported vault provider", zap.Any("provider", cfg.Vault.Provider))
		return "", fmt.Errorf("unsupported vault provider: %s", cfg.Vault.Provider)
	}
}
