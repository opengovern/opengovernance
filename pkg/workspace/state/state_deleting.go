package state

import (
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/transactions"
)

type Deleting struct {
}

func (s Deleting) Requirements() []transactions.TransactionID {
	return []transactions.TransactionID{}
}

func (s Deleting) ProcessingStateID() StateID {
	return StateID_Deleting
}

func (s Deleting) FinishedStateID() StateID {
	return StateID_Deleted
}
