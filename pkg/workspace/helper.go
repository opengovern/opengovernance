package workspace

import (
	"fmt"
	"github.com/labstack/echo/v4"
	authApi "github.com/opengovern/og-util/pkg/api"
	"github.com/opengovern/og-util/pkg/httpclient"
	"github.com/opengovern/og-util/pkg/httpserver"
	"net/http"
)

func (s *Server) CheckRoleInWorkspace(ctx echo.Context, workspaceID, ownerID *string, workspaceName string) error {
	role := httpserver.GetUserRole(ctx)
	if role == authApi.InternalRole {
		return nil
	}

	resp, err := s.authClient.GetUserRoleBindings(httpclient.FromEchoContext(ctx))
	if err != nil {
		return fmt.Errorf("GetUserRoleBindings: %v", err)
	}

	if workspaceID == nil {
		wid := httpserver.GetWorkspaceID(ctx)
		workspaceID = &wid
	}

	hasRoleInWorkspace := false
	for _, roleBinding := range resp.RoleBindings {
		if roleBinding.WorkspaceID == *workspaceID {
			hasRoleInWorkspace = true
		}
	}
	if resp.GlobalRoles != nil {
		hasRoleInWorkspace = true
	}

	if ownerID != nil && httpserver.GetUserID(ctx) == *ownerID {
		return nil
	}

	if !hasRoleInWorkspace {
		return echo.NewHTTPError(http.StatusForbidden, "operation is forbidden")
	}
	return nil
}
