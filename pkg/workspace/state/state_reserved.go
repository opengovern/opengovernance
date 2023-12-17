package state

import (
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
)

type Reserved struct {
}

func (s Reserved) Requirements() []api.TransactionID {
	return []api.TransactionID{
		api.Transaction_CreateInsightBucket,
		api.Transaction_CreateMasterCredential,
		api.Transaction_CreateServiceAccountRoles,
		api.Transaction_CreateOpenSearch,
		api.Transaction_CreateHelmRelease,
	}
}

func (s Reserved) ProcessingStateID() api.StateID {
	return api.StateID_Reserving
}

func (s Reserved) FinishedStateID() api.StateID {
	return api.StateID_Reserved
}
