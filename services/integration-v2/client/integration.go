package client

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/opengovern/og-util/pkg/httpclient"
	"github.com/opengovern/opengovernance/services/integration-v2/api/models"
	"net/http"
)

type IntegrationServiceClient interface {
	GetIntegration(ctx *httpclient.Context, integrationID string) (*models.Integration, error)
}

type integrationClient struct {
	baseURL string
}

func NewIntegrationServiceClient(baseURL string) IntegrationServiceClient {
	return &integrationClient{baseURL: baseURL}
}

func (c *integrationClient) GetIntegration(ctx *httpclient.Context, integrationID string) (*models.Integration, error) {
	url := fmt.Sprintf("%s/api/v1/integrations/%s", c.baseURL, integrationID)
	var response *models.Integration

	if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return response, nil
}
