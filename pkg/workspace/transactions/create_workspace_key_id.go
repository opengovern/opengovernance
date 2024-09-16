package transactions

import (
	"crypto/rand"
	"fmt"
	"github.com/kaytu-io/kaytu-util/pkg/vault"
	"github.com/kaytu-io/open-governance/pkg/workspace/api"
	"github.com/kaytu-io/open-governance/pkg/workspace/config"
	"github.com/kaytu-io/open-governance/pkg/workspace/db"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

type CreateWorkspaceKeyId struct {
	logger      *zap.Logger
	cfg         config.Config
	workspaceDb *db.Database

	vaultSecretHandler vault.VaultSecretHandler
}

func NewCreateWorkspaceKeyId(logger *zap.Logger, vaultSecretHandler vault.VaultSecretHandler, cfg config.Config, workspaceDb *db.Database) *CreateWorkspaceKeyId {
	return &CreateWorkspaceKeyId{
		logger:             logger,
		cfg:                cfg,
		workspaceDb:        workspaceDb,
		vaultSecretHandler: vaultSecretHandler,
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
	case vault.AzureKeyVault, vault.HashiCorpVault:
		// create new aes key
		b := make([]byte, 32)
		_, err := rand.Read(b)
		if err != nil {
			t.logger.Error("failed to generate random bytes", zap.Error(err))
			return err
		}
		name := fmt.Sprintf("client-creds-key-%s", workspace.ID)
		_, err = t.vaultSecretHandler.SetSecret(ctx, name, b)
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
		err := t.vaultSecretHandler.DeleteSecret(ctx, workspace.VaultKeyId)
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
