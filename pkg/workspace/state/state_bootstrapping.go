package state

import "github.com/kaytu-io/kaytu-engine/pkg/workspace/transactions"

type Bootstrapping struct {
}

func (s Bootstrapping) Requirements() []transactions.TransactionID {
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

func (s Bootstrapping) ProcessingStateID() StateID {
	return StateID_Bootstrapping
}

func (s Bootstrapping) FinishedStateID() StateID {
	return StateID_Provisioned
}
