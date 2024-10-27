package client

import (
	"fmt"
	"github.com/labstack/echo/v4"
	authApi "github.com/opengovern/og-util/pkg/api"
	"github.com/opengovern/og-util/pkg/httpclient"
	"github.com/opengovern/opengovernance/services/integration-v2/api/models"
	integration_type "github.com/opengovern/opengovernance/services/integration-v2/integration-type"
	"net/http"
)

type IntegrationServiceClient interface {
	GetIntegration(ctx *httpclient.Context, integrationID string) (*models.Integration, error)
	ListIntegrations(ctx *httpclient.Context, integrationTypes []integration_type.IntegrationType) ([]models.Integration, error)
	IntegrationHealthcheck(ctx *httpclient.Context, integrationID string) (*models.Integration, error)
	GetCredential(ctx *httpclient.Context, credentialID string) (*models.Credential, error)
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

func (c *integrationClient) ListIntegrations(ctx *httpclient.Context, integrationTypes []integration_type.IntegrationType) ([]models.Integration, error) {
	ctx.UserRole = authApi.AdminRole
	url := fmt.Sprintf("%s/api/v1/integrations", c.baseURL)
	for i, v := range integrationTypes {
		if i == 0 {
			url += "?"
		} else {
			url += "&"
		}
		url += "integration_type=" + string(v)
	}

	var response []models.Integration
	if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return response, nil
}

func (c *integrationClient) GetCredential(ctx *httpclient.Context, credentialID string) (*models.Credential, error) {
	url := fmt.Sprintf("%s/api/v1/credentials/%s", c.baseURL, credentialID)
	var response *models.Credential

	if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return response, nil
}

func (c *integrationClient) IntegrationHealthcheck(ctx *httpclient.Context, integrationID string) (*models.Integration, error) {
	url := fmt.Sprintf("%s/api/v1/integrations/%s/healthcheck", c.baseURL, integrationID)
	var response *models.Integration

	if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodPut, url, ctx.ToHeaders(), nil, &response); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return response, nil
}
