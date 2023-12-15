package transactions

import (
	"fmt"
	authapi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	authclient "github.com/kaytu-io/kaytu-engine/pkg/auth/client"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/types"
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

func (t *CreateRoleBinding) ID() types.TransactionID {
	return types.Transaction_CreateRoleBinding
}

func (t *CreateRoleBinding) Requirements() []types.TransactionID {
	return []types.TransactionID{types.Transaction_CreateHelmRelease}
}

func (t *CreateRoleBinding) Apply(workspace db.Workspace) error {
	authCtx := &httpclient.Context{
		UserID:        *workspace.OwnerId,
		UserRole:      authapi.AdminRole,
		WorkspaceName: workspace.Name,
		WorkspaceID:   workspace.ID,
	}

	if err := t.authClient.PutRoleBinding(authCtx, &authapi.PutRoleBindingRequest{
		UserID:   *workspace.OwnerId,
		RoleName: authapi.AdminRole,
	}); err != nil {
		return fmt.Errorf("PutRoleBinding: %w", err)
	}

	return nil
}

func (t *CreateRoleBinding) Rollback(workspace db.Workspace) error {
	//authCtx := &httpclient.Context{
	//	UserID:        *workspace.OwnerId,
	//	UserRole:      authapi.AdminRole,
	//	WorkspaceName: workspace.Name,
	//	WorkspaceID:   workspace.ID,
	//}
	//
	//if err := t.authClient.DeleteRoleBinding(authCtx, &authapi.PutRoleBindingRequest{
	//	UserID:   *workspace.OwnerId,
	//	RoleName: authapi.AdminRole,
	//}); err != nil {
	//	return fmt.Errorf("DeleteRoleBinding: %w", err)
	//}
	return nil
}
