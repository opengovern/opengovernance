package state

import (
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/types"
)

type Bootstrapping struct {
}

func (s Bootstrapping) Requirements() []types.TransactionID {
	return []types.TransactionID{
		types.Transaction_CreateInsightBucket,
		types.Transaction_CreateMasterCredential,
		types.Transaction_CreateServiceAccountRoles,
		types.Transaction_CreateOpenSearch,
		types.Transaction_CreateHelmRelease,
		types.Transaction_CreateRoleBinding,
		types.Transaction_EnsureCredentialOnboarded,
		types.Transaction_EnsureBootstrapInputFinished,
		types.Transaction_EnsureDiscoveryFinished,
		types.Transaction_EnsureJobsRunning,
		types.Transaction_EnsureJobsFinished,
	}
}

func (s Bootstrapping) ProcessingStateID() types.StateID {
	return types.StateID_Bootstrapping
}

func (s Bootstrapping) FinishedStateID() types.StateID {
	return types.StateID_Provisioned
}
