package state

import (
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
)

type Deleting struct {
}

func (s Deleting) Requirements() []api.TransactionID {
	return []api.TransactionID{}
}

func (s Deleting) ProcessingStateID() api.StateID {
	return api.StateID_Deleting
}

func (s Deleting) FinishedStateID() api.StateID {
	return api.StateID_Deleted
}
