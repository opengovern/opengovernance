package transactions

import (
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/types"
)

type EnsureBootstrapInputFinished struct {
}

func NewEnsureBootstrapInputFinished() *EnsureBootstrapInputFinished {
	return &EnsureBootstrapInputFinished{}
}

func (t *EnsureBootstrapInputFinished) Requirements() []types.TransactionID {
	return []types.TransactionID{types.Transaction_EnsureCredentialOnboarded, types.Transaction_CreateHelmRelease}
}

func (t *EnsureBootstrapInputFinished) Apply(workspace db.Workspace) error {
	if workspace.IsBootstrapInputFinished {
		return nil
	}

	return types.ErrTransactionNeedsTime
}

func (t *EnsureBootstrapInputFinished) Rollback(workspace db.Workspace) error {
	return nil
}
