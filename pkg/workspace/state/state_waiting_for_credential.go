package state

import (
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"go.uber.org/zap"
)

type WaitingForCredential struct {
	db     *db.Database
	logger *zap.Logger
}

func (s WaitingForCredential) Requirements(workspace db.Workspace) []api.TransactionID {
	creds, err := s.db.ListCredentialsByWorkspaceID(workspace.ID)
	if err != nil {
		s.logger.Error("failed to list credentials", zap.Error(err), zap.String("workspace_id", workspace.ID))
	}

	if len(creds) == 0 {
		return []api.TransactionID{
			api.Transaction_CreateWorkspaceKeyId,
			api.Transaction_CreateMasterCredential,
			api.Transaction_EnsureCredentialExists,
		}
	}

	return []api.TransactionID{
		api.Transaction_CreateWorkspaceKeyId,
		//api.Transaction_CreateInsightBucket,
		api.Transaction_CreateMasterCredential,
		api.Transaction_CreateServiceAccountRoles,
		//api.Transaction_CreateOpenSearch,
		//api.Transaction_CreateIngestionPipeline,
		api.Transaction_CreateHelmRelease,
		api.Transaction_EnsureCredentialExists,
	}
}

func (s WaitingForCredential) ProcessingStateID() api.StateID {
	return api.StateID_WaitingForCredential
}

func (s WaitingForCredential) FinishedStateID() api.StateID {
	return api.StateID_Provisioning
}
