package state

import (
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
)

type Provisioning struct {
}

func (s Provisioning) Requirements(workspace db.Workspace) []api.TransactionID {
	return []api.TransactionID{
		api.Transaction_CreateWorkspaceKeyId,
		api.Transaction_CreateMasterCredential,
		api.Transaction_CreateServiceAccountRoles,
		api.Transaction_CreateHelmRelease,
		api.Transaction_CreateRoleBinding,
		api.Transaction_EnsureCredentialOnboarded,
		api.Transaction_EnsureDiscoveryFinished,
		api.Transaction_EnsureJobsRunning,
		api.Transaction_EnsureJobsFinished,
	}
}

func (s Provisioning) ProcessingStateID() api.StateID {
	return api.StateID_Provisioning
}

func (s Provisioning) FinishedStateID() api.StateID {
	return api.StateID_Provisioned
}
