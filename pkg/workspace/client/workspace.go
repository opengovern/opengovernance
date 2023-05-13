package client

import (
	"fmt"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
	"net/http"

	"gitlab.com/keibiengine/keibi-engine/pkg/workspace/api"
)

type WorkspaceServiceClient interface {
	GetLimits(ctx *httpclient.Context, workspaceName string) (api.WorkspaceLimitsUsage, error)
	GetLimitsByID(ctx *httpclient.Context, workspaceID string) (api.WorkspaceLimits, error)
	GetByID(ctx *httpclient.Context, workspaceID string) (api.Workspace, error)
	ListWorkspaces(ctx *httpclient.Context) ([]api.WorkspaceResponse, error)
}

type workspaceClient struct {
	baseURL string
}

func NewWorkspaceClient(baseURL string) WorkspaceServiceClient {
	return &workspaceClient{baseURL: baseURL}
}

func (s *workspaceClient) GetLimits(ctx *httpclient.Context, workspaceName string) (api.WorkspaceLimitsUsage, error) {
	url := fmt.Sprintf("%s/api/v1/workspaces/limits/%s", s.baseURL, workspaceName)

	fmt.Println(url)
	var response api.WorkspaceLimitsUsage
	if err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		return api.WorkspaceLimitsUsage{}, err
	}
	return response, nil
}

func (s *workspaceClient) GetLimitsByID(ctx *httpclient.Context, workspaceID string) (api.WorkspaceLimits, error) {
	url := fmt.Sprintf("%s/api/v1/workspaces/limits/byid/%s", s.baseURL, workspaceID)

	var response api.WorkspaceLimits
	if err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		return api.WorkspaceLimits{}, err
	}
	return response, nil
}

func (s *workspaceClient) GetByID(ctx *httpclient.Context, workspaceID string) (api.Workspace, error) {
	url := fmt.Sprintf("%s/api/v1/workspaces/byid/%s", s.baseURL, workspaceID)

	var response api.Workspace
	if err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		return api.Workspace{}, err
	}
	return response, nil
}

func (s *workspaceClient) ListWorkspaces(ctx *httpclient.Context) ([]api.WorkspaceResponse, error) {
	url := fmt.Sprintf("%s/workspace/api/v1/workspaces", s.baseURL)

	var response []api.WorkspaceResponse
	if err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		return nil, err
	}
	return response, nil
}
