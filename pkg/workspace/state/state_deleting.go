package state

import (
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/transactions"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/types"
)

type Deleting struct {
}

func (s Deleting) Requirements() []transactions.TransactionID {
	return []transactions.TransactionID{}
}

func (s Deleting) ProcessingStateID() types.StateID {
	return types.StateID_Deleting
}

func (s Deleting) FinishedStateID() types.StateID {
	return types.StateID_Deleted
}
