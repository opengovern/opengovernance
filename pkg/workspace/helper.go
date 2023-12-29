package workspace

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	"github.com/labstack/echo/v4"
	"net/http"
)

func (s *Server) CheckRoleInWorkspace(ctx echo.Context, ownerID *string) error {
	resp, err := s.authClient.GetUserRoleBindings(httpclient.FromEchoContext(ctx))
	if err != nil {
		return fmt.Errorf("GetUserRoleBindings: %v", err)
	}

	hasRoleInWorkspace := false
	for _, roleBinding := range resp.RoleBindings {
		if roleBinding.WorkspaceID == httpserver.GetWorkspaceID(ctx) {
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
