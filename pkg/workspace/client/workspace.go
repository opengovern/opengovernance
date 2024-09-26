package client

import (
	"fmt"
	"github.com/kaytu-io/kaytu-util/pkg/httpclient"
	"net/http"

	"github.com/kaytu-io/open-governance/pkg/workspace/api"
)

type WorkspaceServiceClient interface {
	GetByName(ctx *httpclient.Context, workspaceName string) (api.Workspace, error)
	ListWorkspaces(ctx *httpclient.Context) ([]api.WorkspaceResponse, error)
	SyncDemo(ctx *httpclient.Context) error
	SetConfiguredStatus(ctx *httpclient.Context) error
	GetConfiguredStatus(ctx *httpclient.Context) (string, error)
}

type workspaceClient struct {
	baseURL string
}

func NewWorkspaceClient(baseURL string) WorkspaceServiceClient {
	return &workspaceClient{baseURL: baseURL}
}

func (s *workspaceClient) GetByName(ctx *httpclient.Context, workspaceName string) (api.Workspace, error) {
	url := fmt.Sprintf("%s/api/v1/workspaces/byname/%s", s.baseURL, workspaceName)
	var response api.Workspace
	if _, err := httpclient.DoRequest(ctx.Ctx, http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		return api.Workspace{}, err
	}
	return response, nil
}

func (s *workspaceClient) ListWorkspaces(ctx *httpclient.Context) ([]api.WorkspaceResponse, error) {
	url := fmt.Sprintf("%s/api/v1/workspaces", s.baseURL)

	var response []api.WorkspaceResponse
	if _, err := httpclient.DoRequest(ctx.Ctx, http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		return nil, err
	}
	return response, nil
}

func (s *workspaceClient) SyncDemo(ctx *httpclient.Context) error {
	url := fmt.Sprintf("%s/api/v3/sample/sync", s.baseURL)

	if _, err := httpclient.DoRequest(ctx.Ctx, http.MethodPut, url, ctx.ToHeaders(), nil, nil); err != nil {
		return err
	}
	return nil
}

func (s *workspaceClient) SetConfiguredStatus(ctx *httpclient.Context) error {
	url := fmt.Sprintf("%s/api/v3/configured/set", s.baseURL)

	if _, err := httpclient.DoRequest(ctx.Ctx, http.MethodPut, url, ctx.ToHeaders(), nil, nil); err != nil {
		return err
	}
	return nil
}

func (s *workspaceClient) GetConfiguredStatus(ctx *httpclient.Context) (string, error) {
	url := fmt.Sprintf("%s/api/v3/configured/status", s.baseURL)

	var status string
	if _, err := httpclient.DoRequest(ctx.Ctx, http.MethodGet, url, ctx.ToHeaders(), nil, &status); err != nil {
		return "", err
	}
	return status, nil
}
