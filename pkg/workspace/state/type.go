package state

import (
	"github.com/opengovern/opengovernance/pkg/workspace/api"
	"github.com/opengovern/opengovernance/pkg/workspace/db"
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
