package client

import (
	"fmt"
	"net/http"

	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"

	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
)

type WorkspaceServiceClient interface {
	GetLimits(ctx *httpclient.Context, workspaceName string, ignoreUsage bool) (api.WorkspaceLimitsUsage, error)
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

func (s *workspaceClient) GetLimits(ctx *httpclient.Context, workspaceName string, ignoreUsage bool) (api.WorkspaceLimitsUsage, error) {
	url := fmt.Sprintf("%s/api/v1/workspaces/limits/%s?ignore_usage=%v", s.baseURL, workspaceName, ignoreUsage)

	var response api.WorkspaceLimitsUsage
	if _, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		return api.WorkspaceLimitsUsage{}, err
	}
	return response, nil
}

func (s *workspaceClient) GetLimitsByID(ctx *httpclient.Context, workspaceID string) (api.WorkspaceLimits, error) {
	url := fmt.Sprintf("%s/api/v1/workspaces/limits/byid/%s", s.baseURL, workspaceID)

	var response api.WorkspaceLimits
	if _, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		return api.WorkspaceLimits{}, err
	}
	return response, nil
}

func (s *workspaceClient) GetByID(ctx *httpclient.Context, workspaceID string) (api.Workspace, error) {
	url := fmt.Sprintf("%s/api/v1/workspaces/byid/%s", s.baseURL, workspaceID)
	var response api.Workspace
	if _, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		return api.Workspace{}, err
	}
	return response, nil
}

func (s *workspaceClient) ListWorkspaces(ctx *httpclient.Context) ([]api.WorkspaceResponse, error) {
	url := fmt.Sprintf("%s/workspace/api/v1/workspaces", s.baseURL)

	var response []api.WorkspaceResponse
	if _, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		return nil, err
	}
	return response, nil
}
