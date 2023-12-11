package httpserver

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/labstack/echo/v4"
)

const (
	XKaytuWorkspaceIDHeader   = "X-Kaytu-WorkspaceID"
	XKaytuWorkspaceNameHeader = "X-Kaytu-WorkspaceName"
	XKaytuUserIDHeader        = "X-Kaytu-UserId"
	XKaytuUserRoleHeader      = "X-Kaytu-UserRole"
)

func AuthorizeHandler(h echo.HandlerFunc, minRole api.Role) echo.HandlerFunc {
	return func(ctx echo.Context) error {
		if err := RequireMinRole(ctx, minRole); err != nil {
			return err
		}

		return h(ctx)
	}
}

func RequireMinRole(ctx echo.Context, minRole api.Role) error {
	if !hasAccess(GetUserRole(ctx), minRole) {
		userRole := GetUserRole(ctx)
		fmt.Println("required role = ", minRole, " user role = ", userRole)
		return echo.NewHTTPError(http.StatusForbidden, "missing required permission")
	}

	return nil
}

func GetWorkspaceName(ctx echo.Context) string {
	name := ctx.Request().Header.Get(XKaytuWorkspaceNameHeader)
	if strings.TrimSpace(name) == "" {
		panic(fmt.Errorf("header %s is missing", XKaytuWorkspaceNameHeader))
	}

	return name
}

func GetWorkspaceID(ctx echo.Context) string {
	id := ctx.Request().Header.Get(XKaytuWorkspaceIDHeader)
	if strings.TrimSpace(id) == "" {
		panic(fmt.Errorf("header %s is missing", XKaytuWorkspaceIDHeader))
	}

	return id
}

func GetUserRole(ctx echo.Context) api.Role {
	role := ctx.Request().Header.Get(XKaytuUserRoleHeader)
	if strings.TrimSpace(role) == "" {
		panic(fmt.Errorf("header %s is missing", XKaytuUserRoleHeader))
	}

	return api.GetRole(role)
}

func GetUserID(ctx echo.Context) string {
	id := ctx.Request().Header.Get(XKaytuUserIDHeader)
	if strings.TrimSpace(id) == "" {
		panic(fmt.Errorf("header %s is missing", XKaytuUserIDHeader))
	}

	return id
}

func roleToPriority(role api.Role) int {
	switch role {
	case api.ViewerRole:
		return 0
	case api.EditorRole:
		return 1
	case api.AdminRole:
		return 2
	case api.KaytuAdminRole:
		return 98
	case api.InternalRole:
		return 99
	default:
		panic("unsupported role: " + role)
	}
}

func hasAccess(currRole, minRole api.Role) bool {
	return roleToPriority(currRole) >= roleToPriority(minRole)
}
