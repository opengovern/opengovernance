package httpserver

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"net/http"
	"strconv"
	"strings"
)

const (
	XKeibiWorkspaceIDHeader   = "X-Keibi-WorkspaceID"
	XKeibiWorkspaceNameHeader = "X-Keibi-WorkspaceName"
	XKeibiUserIDHeader        = "X-Keibi-UserId"
	XKeibiUserRoleHeader      = "X-Keibi-UserRole"

	XKeibiMaxUsersHeader       = "X-Keibi-MaxUsers"
	XKeibiMaxConnectionsHeader = "X-Keibi-MaxConnections"
	XKeibiMaxResourcesHeader   = "X-Keibi-MaxResources"
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
		return echo.NewHTTPError(http.StatusForbidden, "missing required permission")
	}

	return nil
}

func GetWorkspaceName(ctx echo.Context) string {
	name := ctx.Request().Header.Get(XKeibiWorkspaceNameHeader)
	if strings.TrimSpace(name) == "" {
		panic(fmt.Errorf("header %s is missing", XKeibiWorkspaceNameHeader))
	}

	return name
}

func GetWorkspaceID(ctx echo.Context) string {
	id := ctx.Request().Header.Get(XKeibiWorkspaceIDHeader)
	if strings.TrimSpace(id) == "" {
		panic(fmt.Errorf("header %s is missing", XKeibiWorkspaceIDHeader))
	}

	return id
}

func GetUserRole(ctx echo.Context) api.Role {
	role := ctx.Request().Header.Get(XKeibiUserRoleHeader)
	if strings.TrimSpace(role) == "" {
		panic(fmt.Errorf("header %s is missing", XKeibiUserRoleHeader))
	}

	return api.GetRole(role)
}

func GetUserID(ctx echo.Context) string {
	id := ctx.Request().Header.Get(XKeibiUserIDHeader)
	if strings.TrimSpace(id) == "" {
		panic(fmt.Errorf("header %s is missing", XKeibiUserIDHeader))
	}

	return id
}

func GetMaxUsers(ctx echo.Context) int64 {
	max := ctx.Request().Header.Get(XKeibiMaxUsersHeader)
	if strings.TrimSpace(max) == "" {
		panic(fmt.Errorf("header %s is missing", XKeibiMaxUsersHeader))
	}

	c, err := strconv.ParseInt(max, 10, 64)
	if err != nil {
		panic(err)
	}
	return c
}

func GetMaxConnections(ctx echo.Context) int64 {
	max := ctx.Request().Header.Get(XKeibiMaxConnectionsHeader)
	if strings.TrimSpace(max) == "" {
		panic(fmt.Errorf("header %s is missing", XKeibiMaxConnectionsHeader))
	}

	c, err := strconv.ParseInt(max, 10, 64)
	if err != nil {
		panic(err)
	}
	return c
}

func GetMaxResources(ctx echo.Context) int64 {
	max := ctx.Request().Header.Get(XKeibiMaxResourcesHeader)
	if strings.TrimSpace(max) == "" {
		panic(fmt.Errorf("header %s is missing", XKeibiMaxResourcesHeader))
	}

	c, err := strconv.ParseInt(max, 10, 64)
	if err != nil {
		panic(err)
	}
	return c
}

func roleToPriority(role api.Role) int {
	switch role {
	case api.ViewerRole:
		return 0
	case api.EditorRole:
		return 1
	case api.AdminRole:
		return 2
	case api.KeibiAdminRole:
		return 99
	default:
		panic("unsupported role: " + role)
	}
}

func hasAccess(currRole, minRole api.Role) bool {
	return roleToPriority(currRole) >= roleToPriority(minRole)
}
