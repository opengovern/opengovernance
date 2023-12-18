package state

import (
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
)

type WaitingForCredential struct {
}

func (s WaitingForCredential) Requirements() []api.TransactionID {
	return []api.TransactionID{
		api.Transaction_CreateMasterCredential,
		api.Transaction_EnsureCredentialExists,
	}
}

func (s WaitingForCredential) ProcessingStateID() api.StateID {
	return api.StateID_WaitingForCredential
}

func (s WaitingForCredential) FinishedStateID() api.StateID {
	return api.StateID_Provisioning
}
