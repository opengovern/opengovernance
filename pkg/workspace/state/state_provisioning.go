package state

import (
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/transactions"
)

type Provisioning struct {
}

func (s Provisioning) Requirements() []transactions.TransactionID {
	return []transactions.TransactionID{
		transactions.Transaction_CreateInsightBucket,
		transactions.Transaction_CreateMasterCredential,
		transactions.Transaction_CreateServiceAccountRoles,
		transactions.Transaction_CreateOpenSearch,
		transactions.Transaction_CreateHelmRelease,
		transactions.Transaction_CreateRoleBinding,
		transactions.Transaction_EnsureCredentialOnboarded,
		transactions.Transaction_EnsureBootstrapInputFinished,
		transactions.Transaction_EnsureDiscoveryFinished,
		transactions.Transaction_EnsureJobsRunning,
		transactions.Transaction_EnsureJobsFinished,
	}
}

func (s Provisioning) ProcessingStateID() StateID {
	return StateID_Provisioning
}

func (s Provisioning) FinishedStateID() StateID {
	return StateID_Provisioned
}
