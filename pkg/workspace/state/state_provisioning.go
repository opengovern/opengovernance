package state

import (
	"github.com/opengovern/opengovernance/pkg/workspace/api"
	"github.com/opengovern/opengovernance/pkg/workspace/db"
)

type Provisioning struct {
}

func (s Provisioning) Requirements(workspace db.Workspace) []api.TransactionID {
	return []api.TransactionID{
		api.Transaction_CreateWorkspaceKeyId,
		api.Transaction_EnsureWorkspacePodsRunning,
		api.Transaction_CreateRoleBinding,
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
