package transactions

import (
	"errors"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
)

type Transaction interface {
	Requirements() []api.TransactionID
	ApplyIdempotent(workspace db.Workspace) error
	RollbackIdempotent(workspace db.Workspace) error
}

var ErrTransactionNeedsTime = errors.New("transaction needs time")
