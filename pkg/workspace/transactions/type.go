package transactions

import (
	"errors"
	"github.com/opengovern/opengovernance/pkg/workspace/api"
	"github.com/opengovern/opengovernance/pkg/workspace/db"
	"golang.org/x/net/context"
)

type Transaction interface {
	Requirements() []api.TransactionID
	ApplyIdempotent(ctx context.Context, workspace db.Workspace) error
	RollbackIdempotent(ctx context.Context, workspace db.Workspace) error
}

var ErrTransactionNeedsTime = errors.New("transaction needs time")
