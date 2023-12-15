package state

import (
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/types"
)

type Reserved struct {
}

func (s Reserved) Requirements() []types.TransactionID {
	return []types.TransactionID{
		types.Transaction_CreateInsightBucket,
		types.Transaction_CreateMasterCredential,
		types.Transaction_CreateServiceAccountRoles,
		types.Transaction_CreateOpenSearch,
		types.Transaction_CreateHelmRelease,
	}
}

func (s Reserved) ProcessingStateID() types.StateID {
	return types.StateID_Reserving
}

func (s Reserved) FinishedStateID() types.StateID {
	return types.StateID_Reserved
}
