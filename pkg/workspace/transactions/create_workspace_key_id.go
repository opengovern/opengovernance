package transactions

import (
	"crypto/rand"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/config"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"github.com/kaytu-io/kaytu-util/pkg/vault"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

type CreateWorkspaceKeyId struct {
	logger             *zap.Logger
	azureSecretHandler *vault.AzureVaultSecretHandler
	cfg                config.Config
	workspaceDb        *db.Database
}

func NewCreateWorkspaceKeyId(logger *zap.Logger, azureSecretHandler *vault.AzureVaultSecretHandler, cfg config.Config, workspaceDb *db.Database) *CreateWorkspaceKeyId {
	return &CreateWorkspaceKeyId{
		logger:             logger,
		azureSecretHandler: azureSecretHandler,
		cfg:                cfg,
		workspaceDb:        workspaceDb,
	}
}

func (t *CreateWorkspaceKeyId) Requirements() []api.TransactionID {
	return nil
}

func (t *CreateWorkspaceKeyId) ApplyIdempotent(ctx context.Context, workspace db.Workspace) error {
	if workspace.VaultKeyId != "" {
		return nil
	}
	switch t.cfg.Vault.Provider {
	case vault.AwsKMS:
		workspace.VaultKeyId = t.cfg.Vault.KeyId
	case vault.AzureKeyVault:
		// create new aes key
		b := make([]byte, 32)
		_, err := rand.Read(b)
		if err != nil {
			t.logger.Error("failed to generate random bytes", zap.Error(err))
			return err
		}
		name := fmt.Sprintf("client-creds-key-%s", workspace.ID)
		_, err = t.azureSecretHandler.SetSecret(ctx, name, b)
		if err != nil {
			t.logger.Error("failed to set secret", zap.Error(err))
			return err
		}
		workspace.VaultKeyId = name
	default:
		t.logger.Error("unsupported vault provider", zap.Any("provider", t.cfg.Vault.Provider))
		return fmt.Errorf("unsupported vault provider: %s", t.cfg.Vault.Provider)
	}
	return t.workspaceDb.UpdateWorkspace(&workspace)
}

func (t *CreateWorkspaceKeyId) RollbackIdempotent(ctx context.Context, workspace db.Workspace) error {
	if workspace.VaultKeyId == "" {
		return nil
	}
	switch t.cfg.Vault.Provider {
	case vault.AwsKMS:
		workspace.VaultKeyId = ""
	case vault.AzureKeyVault:
		err := t.azureSecretHandler.DeleteSecret(ctx, workspace.VaultKeyId)
		if err != nil {
			t.logger.Error("failed to delete secret", zap.Error(err))
			return err
		}
		workspace.VaultKeyId = ""
	default:
		t.logger.Error("unsupported vault provider", zap.Any("provider", t.cfg.Vault.Provider))
		return fmt.Errorf("unsupported vault provider: %s", t.cfg.Vault.Provider)
	}
	return t.workspaceDb.UpdateWorkspace(&workspace)
}
