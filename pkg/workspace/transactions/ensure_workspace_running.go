package transactions

import (
	"context"
	authApi "github.com/kaytu-io/kaytu-util/pkg/api"
	"github.com/kaytu-io/kaytu-util/pkg/httpclient"
	"github.com/kaytu-io/kaytu-util/pkg/vault"
	"github.com/kaytu-io/open-governance/pkg/compliance/client"
	"github.com/kaytu-io/open-governance/pkg/workspace/api"
	"github.com/kaytu-io/open-governance/pkg/workspace/config"
	"github.com/kaytu-io/open-governance/pkg/workspace/db"
	"go.uber.org/zap"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type EnsureWorkspacePodsRunning struct {
	kubeClient         k8sclient.Client // the kubernetes client
	vault              vault.VaultSourceConfig
	cfg                config.Config
	db                 *db.Database
	logger             *zap.Logger
	vaultSecretHandler vault.VaultSecretHandler
}

func NewEnsureWorkspacePodsRunning(kubeClient k8sclient.Client, vault vault.VaultSourceConfig, handler vault.VaultSecretHandler, cfg config.Config, db *db.Database, logger *zap.Logger) *EnsureWorkspacePodsRunning {
	return &EnsureWorkspacePodsRunning{
		kubeClient:         kubeClient,
		vaultSecretHandler: handler,
		vault:              vault,
		cfg:                cfg,
		db:                 db,
		logger:             logger,
	}
}

func (t *EnsureWorkspacePodsRunning) Requirements() []api.TransactionID {
	return []api.TransactionID{
		api.Transaction_CreateWorkspaceKeyId,
	}
}

func (t *EnsureWorkspacePodsRunning) ApplyIdempotent(ctx context.Context, workspace db.Workspace) error {
	complianceURL := strings.ReplaceAll(t.cfg.Compliance.BaseURL, "%NAMESPACE%", t.cfg.KaytuOctopusNamespace)
	complianceClient := client.NewComplianceClient(complianceURL)

	_, err := complianceClient.CountFindings(&httpclient.Context{
		Ctx:      ctx,
		UserRole: authApi.InternalRole,
	}, nil)
	if err != nil {
		return ErrTransactionNeedsTime
	}

	err = t.db.SetWorkspaceCreated(workspace.ID)
	if err != nil {
		t.logger.Error("failed to set workspace created", zap.Error(err))
		return err
	}

	return nil
}

func (t *EnsureWorkspacePodsRunning) RollbackIdempotent(ctx context.Context, workspace db.Workspace) error {
	return nil
}
