package state

import (
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/transactions"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/types"
)

type Reserved struct {
}

func (s Reserved) Requirements() []transactions.TransactionID {
	return []transactions.TransactionID{
		transactions.Transaction_CreateInsightBucket,
		transactions.Transaction_CreateMasterCredential,
		transactions.Transaction_CreateServiceAccountRoles,
		transactions.Transaction_CreateOpenSearch,
		transactions.Transaction_CreateHelmRelease,
	}
}

func (s Reserved) ProcessingStateID() types.StateID {
	return types.StateID_Reserving
}

func (s Reserved) FinishedStateID() types.StateID {
	return types.StateID_Reserved
}
