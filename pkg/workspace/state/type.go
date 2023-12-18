package state

import (
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
)

type State interface {
	Requirements() []api.TransactionID
	ProcessingStateID() api.StateID
	FinishedStateID() api.StateID
}

var AllStates = []State{
	WaitingForCredential{},
	Provisioning{},
	Deleting{},
	Reserved{},
}
