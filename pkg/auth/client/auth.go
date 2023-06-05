package client

import (
	"encoding/json"
	"fmt"
	"net/http"

	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"

	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
)

type AuthServiceClient interface {
	PutRoleBinding(ctx *httpclient.Context, request *api.PutRoleBindingRequest) error
	GetWorkspaceRoleBindings(ctx *httpclient.Context, workspaceName, workspaceID string) (api.GetWorkspaceRoleBindingResponse, error)
	GetUserRoleBindings(ctx *httpclient.Context) (api.GetRoleBindingsResponse, error)
}

type authClient struct {
	baseURL string
}

func NewAuthServiceClient(baseURL string) AuthServiceClient {
	return &authClient{baseURL: baseURL}
}

func (c *authClient) PutRoleBinding(ctx *httpclient.Context, request *api.PutRoleBindingRequest) error {
	url := fmt.Sprintf("%s/api/v1/user/role/binding", c.baseURL)

	payload, err := json.Marshal(api.PutRoleBindingRequest{
		UserID:   request.UserID,
		RoleName: request.RoleName,
	})
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}

	headers := map[string]string{
		httpserver.XKeibiUserIDHeader:        ctx.UserID,
		httpserver.XKeibiUserRoleHeader:      string(ctx.UserRole),
		httpserver.XKeibiWorkspaceNameHeader: ctx.WorkspaceName,
		httpserver.XKeibiWorkspaceIDHeader:   ctx.WorkspaceID,

		httpserver.XKeibiMaxUsersHeader:       fmt.Sprintf("%d", ctx.MaxUsers),
		httpserver.XKeibiMaxConnectionsHeader: fmt.Sprintf("%d", ctx.MaxConnections),
		httpserver.XKeibiMaxResourcesHeader:   fmt.Sprintf("%d", ctx.MaxResources),
	}
	_, res := httpclient.DoRequest(http.MethodPut, url, headers, payload, nil)
	return res
}

func (c *authClient) GetWorkspaceRoleBindings(ctx *httpclient.Context, workspaceName, workspaceID string) (api.GetWorkspaceRoleBindingResponse, error) {
	url := fmt.Sprintf("%s/api/v1/workspace/role/bindings", c.baseURL)

	headers := map[string]string{
		httpserver.XKeibiUserIDHeader:        ctx.UserID,
		httpserver.XKeibiUserRoleHeader:      string(ctx.UserRole),
		httpserver.XKeibiWorkspaceNameHeader: workspaceName,
		httpserver.XKeibiWorkspaceIDHeader:   workspaceID,

		httpserver.XKeibiMaxUsersHeader:       fmt.Sprintf("%d", ctx.MaxUsers),
		httpserver.XKeibiMaxConnectionsHeader: fmt.Sprintf("%d", ctx.MaxConnections),
		httpserver.XKeibiMaxResourcesHeader:   fmt.Sprintf("%d", ctx.MaxResources),
	}
	var response api.GetWorkspaceRoleBindingResponse
	_, err := httpclient.DoRequest(http.MethodGet, url, headers, nil, &response)
	return response, err
}

func (c *authClient) GetUserRoleBindings(ctx *httpclient.Context) (api.GetRoleBindingsResponse, error) {
	url := fmt.Sprintf("%s/api/v1/user/role/bindings", c.baseURL)

	headers := map[string]string{
		httpserver.XKeibiUserIDHeader:        ctx.UserID,
		httpserver.XKeibiUserRoleHeader:      string(ctx.UserRole),
		httpserver.XKeibiWorkspaceNameHeader: ctx.WorkspaceName,
		httpserver.XKeibiWorkspaceIDHeader:   ctx.WorkspaceID,

		httpserver.XKeibiMaxUsersHeader:       fmt.Sprintf("%d", ctx.MaxUsers),
		httpserver.XKeibiMaxConnectionsHeader: fmt.Sprintf("%d", ctx.MaxConnections),
		httpserver.XKeibiMaxResourcesHeader:   fmt.Sprintf("%d", ctx.MaxResources),
	}
	var response api.GetRoleBindingsResponse
	_, err := httpclient.DoRequest(http.MethodGet, url, headers, nil, &response)
	return response, err
}
