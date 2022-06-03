package client

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/httprequest"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpserver"
)

type Context struct {
	UserID        uuid.UUID
	UserRole      api.Role
	WorkspaceName string
}

type AuthServiceClient interface {
	PutRoleBinding(ctx *Context, request *api.PutRoleBindingRequest) error
}

type authClient struct {
	baseURL string
}

func NewAuthServiceClient(baseURL string) AuthServiceClient {
	return &authClient{baseURL: baseURL}
}

func (c *authClient) PutRoleBinding(ctx *Context, request *api.PutRoleBindingRequest) error {
	url := fmt.Sprintf("%s/api/v1/user/role/binding", c.baseURL)

	payload, err := json.Marshal(api.PutRoleBindingRequest{
		UserID: request.UserID,
		Role:   request.Role,
	})
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}

	headers := map[string]string{
		httpserver.XKeibiUserIDHeader:        ctx.UserID.String(),
		httpserver.XKeibiUserRoleHeader:      string(ctx.UserRole),
		httpserver.XKeibiWorkspaceNameHeader: ctx.WorkspaceName,
	}
	return httprequest.DoRequest(http.MethodPut, url, headers, payload, nil)
}
