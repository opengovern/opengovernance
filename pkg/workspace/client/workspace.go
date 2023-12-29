package client

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	"net/http"

	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
)

type WorkspaceServiceClient interface {
	GetByID(ctx *httpclient.Context, workspaceID string) (api.Workspace, error)
	ListWorkspaces(ctx *httpclient.Context) ([]api.WorkspaceResponse, error)
}

type workspaceClient struct {
	baseURL string
}

func NewWorkspaceClient(baseURL string) WorkspaceServiceClient {
	return &workspaceClient{baseURL: baseURL}
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
	url := fmt.Sprintf("%s/api/v1/workspaces", s.baseURL)

	var response []api.WorkspaceResponse
	if _, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		return nil, err
	}
	return response, nil
}
