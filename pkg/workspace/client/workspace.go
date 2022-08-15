package client

import (
	"fmt"
	"net/http"

	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
	"gitlab.com/keibiengine/keibi-engine/pkg/workspace/api"
)

type WorkspaceServiceClient interface {
	GetLimits(ctx *httpclient.Context) (api.WorkspaceLimits, error)
}

type workspaceClient struct {
	baseURL string
}

func NewWorkspaceClient(baseURL string) WorkspaceServiceClient {
	return &workspaceClient{baseURL: baseURL}
}

func (s *workspaceClient) GetLimits(ctx *httpclient.Context) (api.WorkspaceLimits, error) {
	url := fmt.Sprintf("%s/api/v1/workspaces/limits/%s", s.baseURL, ctx.WorkspaceName)

	var response api.WorkspaceLimits
	if err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		return api.WorkspaceLimits{}, err
	}
	return response, nil
}
