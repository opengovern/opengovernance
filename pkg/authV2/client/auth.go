package client

import (
	"encoding/json"
	"fmt"
	"github.com/opengovern/og-util/pkg/httpclient"
	"github.com/opengovern/og-util/pkg/httpserver"
	"net/http"

	"github.com/opengovern/opengovernance/pkg/auth/api"
)

type AuthServiceClient interface {
	PutRoleBinding(ctx *httpclient.Context, request *api.PutRoleBindingRequest) error
	GetWorkspaceRoleBindings(ctx *httpclient.Context, workspaceID string) (api.GetWorkspaceRoleBindingResponse, error)
	GetUserRoleBindings(ctx *httpclient.Context) (api.GetRoleBindingsResponse, error)
	ListAPIKeys(ctx *httpclient.Context, workspaceID string) ([]api.WorkspaceApiKey, error)
	UpdateWorkspaceMap(ctx *httpclient.Context) error
	DeleteRoleBinding(ctx *httpclient.Context, workspaceID, userID string) error
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
		httpserver.XKaytuUserIDHeader:        ctx.UserID,
		httpserver.XKaytuUserRoleHeader:      string(ctx.UserRole),
		httpserver.XKaytuWorkspaceNameHeader: ctx.WorkspaceName,
		httpserver.XKaytuWorkspaceIDHeader:   ctx.WorkspaceID,
	}
	_, res := httpclient.DoRequest(ctx.Ctx, http.MethodPut, url, headers, payload, nil)
	return res
}

func (c *authClient) DeleteRoleBinding(ctx *httpclient.Context, workspaceID, userID string) error {
	url := fmt.Sprintf("%s/api/v1/user/role/binding?userId=%s", c.baseURL, userID)

	headers := map[string]string{
		httpserver.XKaytuUserIDHeader:      ctx.UserID,
		httpserver.XKaytuUserRoleHeader:    string(ctx.UserRole),
		httpserver.XKaytuWorkspaceIDHeader: workspaceID,
	}
	_, res := httpclient.DoRequest(ctx.Ctx, http.MethodDelete, url, headers, nil, nil)
	return res
}

func (c *authClient) GetWorkspaceRoleBindings(ctx *httpclient.Context, workspaceID string) (api.GetWorkspaceRoleBindingResponse, error) {
	url := fmt.Sprintf("%s/api/v1/workspace/role/bindings", c.baseURL)

	headers := map[string]string{
		httpserver.XKaytuUserIDHeader:      ctx.UserID,
		httpserver.XKaytuUserRoleHeader:    string(ctx.UserRole),
		httpserver.XKaytuWorkspaceIDHeader: workspaceID,
	}
	var response api.GetWorkspaceRoleBindingResponse
	_, err := httpclient.DoRequest(ctx.Ctx, http.MethodGet, url, headers, nil, &response)
	return response, err
}

func (c *authClient) ListAPIKeys(ctx *httpclient.Context, workspaceID string) ([]api.WorkspaceApiKey, error) {
	url := fmt.Sprintf("%s/api/v1/keys", c.baseURL)

	headers := map[string]string{
		httpserver.XKaytuUserIDHeader:      ctx.UserID,
		httpserver.XKaytuUserRoleHeader:    string(ctx.UserRole),
		httpserver.XKaytuWorkspaceIDHeader: workspaceID,
	}
	var response []api.WorkspaceApiKey
	_, err := httpclient.DoRequest(ctx.Ctx, http.MethodGet, url, headers, nil, &response)
	return response, err
}

func (c *authClient) GetUserRoleBindings(ctx *httpclient.Context) (api.GetRoleBindingsResponse, error) {
	url := fmt.Sprintf("%s/api/v1/user/role/bindings", c.baseURL)

	headers := map[string]string{
		httpserver.XKaytuUserIDHeader:        ctx.UserID,
		httpserver.XKaytuUserRoleHeader:      string(ctx.UserRole),
		httpserver.XKaytuWorkspaceNameHeader: ctx.WorkspaceName,
		httpserver.XKaytuWorkspaceIDHeader:   ctx.WorkspaceID,
	}
	var response api.GetRoleBindingsResponse
	_, err := httpclient.DoRequest(ctx.Ctx, http.MethodGet, url, headers, nil, &response)
	return response, err
}

func (c *authClient) UpdateWorkspaceMap(ctx *httpclient.Context) error {
	url := fmt.Sprintf("%s/api/v1/workspace-map/update", c.baseURL)

	headers := map[string]string{
		httpserver.XKaytuUserIDHeader:        ctx.UserID,
		httpserver.XKaytuUserRoleHeader:      string(ctx.UserRole),
		httpserver.XKaytuWorkspaceNameHeader: ctx.WorkspaceName,
		httpserver.XKaytuWorkspaceIDHeader:   ctx.WorkspaceID,
	}
	_, err := httpclient.DoRequest(ctx.Ctx, http.MethodPost, url, headers, nil, nil)
	return err
}
