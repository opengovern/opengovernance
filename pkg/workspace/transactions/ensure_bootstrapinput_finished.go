package transactions

import (
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
)

type EnsureBootstrapInputFinished struct {
}

func NewEnsureBootstrapInputFinished() *EnsureBootstrapInputFinished {
	return &EnsureBootstrapInputFinished{}
}

func (t *EnsureBootstrapInputFinished) Requirements() []TransactionID {
	return []TransactionID{Transaction_EnsureCredentialOnboarded, Transaction_CreateHelmRelease}
}

func (t *EnsureBootstrapInputFinished) Apply(workspace db.Workspace) error {
	if workspace.IsBootstrapInputFinished {
		return nil
	}

	return ErrTransactionNeedsTime
}

func (t *EnsureBootstrapInputFinished) Rollback(workspace db.Workspace) error {
	return nil
}
