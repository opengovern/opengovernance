package client

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"

	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
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
		httpserver.XKaytuUserIDHeader:         ctx.UserID,
		httpserver.XKaytuUserRoleHeader:       string(ctx.UserRole),
		httpserver.XKaytuWorkspaceNameHeader:  ctx.WorkspaceName,
		httpserver.XKaytuWorkspaceIDHeader:    ctx.WorkspaceID,
		httpserver.XKaytuMaxUsersHeader:       fmt.Sprintf("%d", ctx.MaxUsers),
		httpserver.XKaytuMaxConnectionsHeader: fmt.Sprintf("%d", ctx.MaxConnections),
		httpserver.XKaytuMaxResourcesHeader:   fmt.Sprintf("%d", ctx.MaxResources),
	}
	_, res := httpclient.DoRequest(http.MethodPut, url, headers, payload, nil)
	return res
}

func (c *authClient) GetWorkspaceRoleBindings(ctx *httpclient.Context, workspaceName, workspaceID string) (api.GetWorkspaceRoleBindingResponse, error) {
	url := fmt.Sprintf("%s/api/v1/workspace/role/bindings", c.baseURL)

	headers := map[string]string{
		httpserver.XKaytuUserIDHeader:        ctx.UserID,
		httpserver.XKaytuUserRoleHeader:      string(ctx.UserRole),
		httpserver.XKaytuWorkspaceNameHeader: workspaceName,
		httpserver.XKaytuWorkspaceIDHeader:   workspaceID,

		httpserver.XKaytuMaxUsersHeader:       fmt.Sprintf("%d", ctx.MaxUsers),
		httpserver.XKaytuMaxConnectionsHeader: fmt.Sprintf("%d", ctx.MaxConnections),
		httpserver.XKaytuMaxResourcesHeader:   fmt.Sprintf("%d", ctx.MaxResources),
	}
	var response api.GetWorkspaceRoleBindingResponse
	_, err := httpclient.DoRequest(http.MethodGet, url, headers, nil, &response)
	return response, err
}

func (c *authClient) GetUserRoleBindings(ctx *httpclient.Context) (api.GetRoleBindingsResponse, error) {
	url := fmt.Sprintf("%s/api/v1/user/role/bindings", c.baseURL)

	headers := map[string]string{
		httpserver.XKaytuUserIDHeader:        ctx.UserID,
		httpserver.XKaytuUserRoleHeader:      string(ctx.UserRole),
		httpserver.XKaytuWorkspaceNameHeader: ctx.WorkspaceName,
		httpserver.XKaytuWorkspaceIDHeader:   ctx.WorkspaceID,

		httpserver.XKaytuMaxUsersHeader:       fmt.Sprintf("%d", ctx.MaxUsers),
		httpserver.XKaytuMaxConnectionsHeader: fmt.Sprintf("%d", ctx.MaxConnections),
		httpserver.XKaytuMaxResourcesHeader:   fmt.Sprintf("%d", ctx.MaxResources),
	}
	var response api.GetRoleBindingsResponse
	_, err := httpclient.DoRequest(http.MethodGet, url, headers, nil, &response)
	return response, err
}
