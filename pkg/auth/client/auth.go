package client

import (
	"encoding/json"
	"fmt"
	"net/http"

	api2 "gitlab.com/keibiengine/keibi-engine/pkg/workspace/api"

	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
)

type AuthServiceClient interface {
	PutRoleBinding(ctx *httpclient.Context, request *api.PutRoleBindingRequest, limits api2.WorkspaceLimits) error
}

type authClient struct {
	baseURL string
}

func NewAuthServiceClient(baseURL string) AuthServiceClient {
	return &authClient{baseURL: baseURL}
}

func (c *authClient) PutRoleBinding(ctx *httpclient.Context, request *api.PutRoleBindingRequest, limits api2.WorkspaceLimits) error {
	url := fmt.Sprintf("%s/api/v1/user/role/binding", c.baseURL)

	payload, err := json.Marshal(api.PutRoleBindingRequest{
		UserID: request.UserID,
		Role:   request.Role,
	})
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}

	headers := map[string]string{
		httpserver.XKeibiUserIDHeader:        ctx.UserID,
		httpserver.XKeibiUserRoleHeader:      string(ctx.UserRole),
		httpserver.XKeibiWorkspaceNameHeader: ctx.WorkspaceName,

		httpserver.XKeibiMaxUsersHeader:       fmt.Sprintf("%d", limits.MaxUsers),
		httpserver.XKeibiMaxConnectionsHeader: fmt.Sprintf("%d", limits.MaxConnections),
		httpserver.XKeibiMaxResourcesHeader:   fmt.Sprintf("%d", limits.MaxResources),
	}
	return httpclient.DoRequest(http.MethodPut, url, headers, payload, nil)
}
