package state

import (
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"go.uber.org/zap"
)

type State interface {
	Requirements(workspace db.Workspace) []api.TransactionID
	ProcessingStateID() api.StateID
	FinishedStateID() api.StateID
}

func AllStates(db *db.Database, logger *zap.Logger) []State {
	return []State{
		Provisioning{},
	}
}
