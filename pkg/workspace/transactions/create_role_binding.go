package transactions

import (
	"fmt"
	api2 "github.com/opengovern/og-util/pkg/api"
	"github.com/opengovern/og-util/pkg/httpclient"
	authapi "github.com/opengovern/opengovernance/pkg/auth/api"
	authclient "github.com/opengovern/opengovernance/pkg/auth/client"
	"github.com/opengovern/opengovernance/pkg/workspace/api"
	"github.com/opengovern/opengovernance/pkg/workspace/db"
	"golang.org/x/net/context"
)

type CreateRoleBinding struct {
	authClient authclient.AuthServiceClient
}

func NewCreateRoleBinding(
	authClient authclient.AuthServiceClient,
) *CreateRoleBinding {
	return &CreateRoleBinding{
		authClient: authClient,
	}
}

func (t *CreateRoleBinding) Requirements() []api.TransactionID {
	return []api.TransactionID{api.Transaction_EnsureWorkspacePodsRunning}
}

func (t *CreateRoleBinding) ApplyIdempotent(ctx context.Context, workspace db.Workspace) error {
	authCtx := &httpclient.Context{
		UserID:        *workspace.OwnerId,
		UserRole:      api2.AdminRole,
		WorkspaceName: workspace.Name,
		WorkspaceID:   workspace.ID,
	}

	if err := t.authClient.PutRoleBinding(authCtx, &authapi.PutRoleBindingRequest{
		UserID:   *workspace.OwnerId,
		RoleName: api2.AdminRole,
	}); err != nil {
		return fmt.Errorf("PutRoleBinding: %w", err)
	}

	return nil
}

func (t *CreateRoleBinding) RollbackIdempotent(ctx context.Context, workspace db.Workspace) error {
	authCtx := &httpclient.Context{
		UserID:        api2.GodUserID,
		UserRole:      api2.InternalRole,
		WorkspaceName: workspace.Name,
		WorkspaceID:   workspace.ID,
	}

	if err := t.authClient.DeleteRoleBinding(authCtx, workspace.ID, *workspace.OwnerId); err != nil {
		return fmt.Errorf("DeleteRoleBinding: %w", err)
	}
	return nil
}
