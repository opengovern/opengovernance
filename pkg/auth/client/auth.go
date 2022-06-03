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

func SetRoleBinding(baseUrl string, userID uuid.UUID, workspaceName string) error {
	url := fmt.Sprintf("%s/api/v1/user/role/binding", baseUrl)

	payload, err := json.Marshal(api.PutRoleBindingRequest{
		UserID: userID,
		Role:   api.AdminRole,
	})
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}

	headers := map[string]string{
		httpserver.XKeibiUserIDHeader:        userID.String(),
		httpserver.XKeibiUserRoleHeader:      string(api.AdminRole),
		httpserver.XKeibiWorkspaceNameHeader: workspaceName,
	}
	return httprequest.DoRequest(http.MethodPut, url, headers, payload, nil)
}
