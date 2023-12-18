package state

import (
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
)

type Deleting struct {
}

func (s Deleting) Requirements(workspace db.Workspace) []api.TransactionID {
	return []api.TransactionID{}
}

func (s Deleting) ProcessingStateID() api.StateID {
	return api.StateID_Deleting
}

func (s Deleting) FinishedStateID() api.StateID {
	return api.StateID_Deleted
}
