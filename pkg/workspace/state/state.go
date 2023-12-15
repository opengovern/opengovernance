package state

import (
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/transactions"
)

type StateID string

const (
	StateID_Reserving     StateID = "RESERVING"
	StateID_Reserved      StateID = "RESERVED"
	StateID_Bootstrapping StateID = "BOOTSTRAPPING"
	StateID_Provisioned   StateID = "PROVISIONED"
	StateID_Deleting      StateID = "DELETING"
	StateID_Deleted       StateID = "DELETED"
)

func (s StateID) IsReserve() bool {
	return s == StateID_Reserving || s == StateID_Reserved
}

type State interface {
	Requirements() []transactions.TransactionID
	ProcessingStateID() StateID
	FinishedStateID() StateID
}

var AllStates = []State{
	Bootstrapping{},
	Deleting{},
	Reserved{},
}
