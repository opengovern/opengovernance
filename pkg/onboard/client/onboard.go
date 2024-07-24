package client

import (
	"context"
	"encoding/json"
	"fmt"
	apiv2 "github.com/kaytu-io/kaytu-engine/pkg/onboard/api/v2"
	authApi "github.com/kaytu-io/kaytu-util/pkg/api"
	"github.com/kaytu-io/kaytu-util/pkg/httpclient"
	kaytuTrace "github.com/kaytu-io/kaytu-util/pkg/trace"
	"go.opentelemetry.io/otel"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/kaytu-io/kaytu-util/pkg/source"

	"github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
)

type OnboardServiceClient interface {
	GetSource(ctx *httpclient.Context, sourceID string) (*api.Connection, error)
	GetSourceFullCred(ctx *httpclient.Context, sourceID string) (*api.AWSCredentialConfig, *api.AzureCredentialConfig, error)
	GetSources(ctx *httpclient.Context, sourceID []string) ([]api.Connection, error)
	ListSources(ctx *httpclient.Context, t []source.Type) ([]api.Connection, error)
	CountSources(ctx *httpclient.Context, provider source.Type) (int64, error)
	PostCredentials(ctx *httpclient.Context, req api.CreateCredentialRequest) (*api.CreateCredentialResponse, error)
	AutoOnboard(ctx *httpclient.Context, credentialId string) ([]api.Connection, error)
	GetSourceHealthcheck(ctx *httpclient.Context, connection string, updateMetadata bool) (*api.Connection, error)
	SetConnectionLifecycleState(ctx *httpclient.Context, connectionId string, state api.ConnectionLifecycleState) (*api.Connection, error)
	ListCredentials(ctx *httpclient.Context, connector []source.Type, credentialType *api.CredentialType, health *string, pageSize, pageNumber int) (api.ListCredentialResponse, error)
	TriggerAutoOnboard(ctx *httpclient.Context, credentialId string) ([]api.Connection, error)
	GetConnectionGroup(ctx *httpclient.Context, connectionGroupName string) (*api.ConnectionGroup, error)
	ListConnectionGroups(ctx *httpclient.Context) ([]api.ConnectionGroup, error)
	CreateCredentialV2(ctx *httpclient.Context, req apiv2.CreateCredentialV2Request) (*apiv2.CreateCredentialV2Response, error)
	PostConnectionAws(ctx *httpclient.Context, req api.CreateAwsConnectionRequest) (*api.CreateConnectionResponse, error)
}

type onboardClient struct {
	baseURL string
}

func NewOnboardServiceClient(baseURL string) OnboardServiceClient {
	return &onboardClient{
		baseURL: baseURL,
	}
}

func (s *onboardClient) PostConnectionAws(ctx *httpclient.Context, req api.CreateAwsConnectionRequest) (*api.CreateConnectionResponse, error) {
	url := fmt.Sprintf("%s/api/v1/connections/aws", s.baseURL)
	var response *api.CreateConnectionResponse

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodPost, url, ctx.ToHeaders(), payload, &response); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return response, nil
}
func (s *onboardClient) GetSource(ctx *httpclient.Context, sourceID string) (*api.Connection, error) {
	if ctx.Ctx == nil {
		ctx.Ctx = context.Background()
	}
	_, span := otel.Tracer(kaytuTrace.JaegerTracerName).Start(ctx.Ctx, kaytuTrace.GetCurrentFuncName())
	defer span.End()

	ctx.UserRole = authApi.InternalRole
	url := fmt.Sprintf("%s/api/v1/source/%s", s.baseURL, sourceID)

	var source api.Connection
	if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodGet, url, ctx.ToHeaders(), nil, &source); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return &source, nil
}

func (s *onboardClient) GetSourceFullCred(ctx *httpclient.Context, sourceID string) (*api.AWSCredentialConfig, *api.AzureCredentialConfig, error) {
	if ctx.Ctx == nil {
		ctx.Ctx = context.Background()
	}
	_, span := otel.Tracer(kaytuTrace.JaegerTracerName).Start(ctx.Ctx, kaytuTrace.GetCurrentFuncName())
	defer span.End()
	url := fmt.Sprintf("%s/api/v1/source/%s/credentials/full", s.baseURL, sourceID)

	var awsCred api.AWSCredentialConfig
	var azureCred api.AzureCredentialConfig

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range ctx.ToHeaders() {
		req.Header.Add(k, v)
	}
	t := http.DefaultTransport.(*http.Transport)
	t.MaxIdleConns = 100
	t.MaxConnsPerHost = 100
	t.MaxIdleConnsPerHost = 100
	client := http.Client{
		Timeout:   15 * time.Second,
		Transport: t,
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("do request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		d, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, nil, fmt.Errorf("read body: %w", err)
		}
		return nil, nil, fmt.Errorf("http status: %d: %s", res.StatusCode, d)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, nil, err
	}

	if err = json.Unmarshal(body, &awsCred); err == nil && awsCred.AccessKey != "" {
		return &awsCred, nil, nil
	}
	if err = json.Unmarshal(body, &azureCred); err == nil && azureCred.ClientId != "" {
		return nil, &azureCred, nil
	}
	return nil, nil, err
}

func (s *onboardClient) GetSources(ctx *httpclient.Context, sourceIDs []string) ([]api.Connection, error) {
	url := fmt.Sprintf("%s/api/v1/sources", s.baseURL)

	var req api.GetSourcesRequest
	var res []api.Connection

	for _, sourceID := range sourceIDs {
		req.SourceIDs = append(req.SourceIDs, sourceID)
	}

	if len(req.SourceIDs) > 0 {
		payload, err := json.Marshal(req)
		if err != nil {
			return nil, err
		}

		var response []api.Connection
		if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodPost, url, ctx.ToHeaders(), payload, &response); err != nil {
			if 400 <= statusCode && statusCode < 500 {
				return nil, echo.NewHTTPError(statusCode, err.Error())
			}
			return nil, err
		}
		res = append(res, response...)
	}

	return res, nil
}

func (s *onboardClient) ListSources(ctx *httpclient.Context, t []source.Type) ([]api.Connection, error) {
	ctx.UserRole = authApi.InternalRole
	url := fmt.Sprintf("%s/api/v1/sources", s.baseURL)
	for i, v := range t {
		if i == 0 {
			url += "?"
		} else {
			url += "&"
		}
		url += "connector=" + string(v)
	}

	var response []api.Connection
	if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return response, nil
}

func (s *onboardClient) PostCredentials(ctx *httpclient.Context, req api.CreateCredentialRequest) (*api.CreateCredentialResponse, error) {
	url := fmt.Sprintf("%s/api/v1/credential", s.baseURL)
	var response *api.CreateCredentialResponse

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodPost, url, ctx.ToHeaders(), payload, &response); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return response, nil
}

func (s *onboardClient) CreateCredentialV2(ctx *httpclient.Context, req apiv2.CreateCredentialV2Request) (*apiv2.CreateCredentialV2Response, error) {
	url := fmt.Sprintf("%s/api/v2/credential", s.baseURL)
	var response *apiv2.CreateCredentialV2Response

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodPost, url, ctx.ToHeaders(), payload, &response); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return response, nil
}

func (s *onboardClient) AutoOnboard(ctx *httpclient.Context, credentialId string) ([]api.Connection, error) {
	url := fmt.Sprintf("%s/api/v1/credential/%s/autoonboard", s.baseURL, credentialId)
	var response []api.Connection

	if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodPost, url, ctx.ToHeaders(), nil, &response); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return response, nil
}

func (s *onboardClient) CountSources(ctx *httpclient.Context, provider source.Type) (int64, error) {
	var url string
	if !provider.IsNull() {
		url = fmt.Sprintf("%s/api/v1/sources/count?connector=%s", s.baseURL, provider.String())
	} else {
		url = fmt.Sprintf("%s/api/v1/sources/count", s.baseURL)
	}

	var count int64
	if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodGet, url, ctx.ToHeaders(), nil, &count); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return 0, echo.NewHTTPError(statusCode, err.Error())
		}
		return 0, err
	}
	return count, nil
}

func (s *onboardClient) GetSourceHealthcheck(ctx *httpclient.Context, connectionId string, updateMetadata bool) (*api.Connection, error) {
	url := fmt.Sprintf("%s/api/v1/source/%s/healthcheck", s.baseURL, connectionId)

	var connection api.Connection

	url += "?"
	//firstParamAttached = true
	url += "updateMetadata=" + strconv.FormatBool(updateMetadata)

	if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodGet, url, ctx.ToHeaders(), nil, &connection); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return &connection, nil
}

func (s *onboardClient) SetConnectionLifecycleState(ctx *httpclient.Context, connectionId string, state api.ConnectionLifecycleState) (*api.Connection, error) {
	url := fmt.Sprintf("%s/api/v1/connections/%s/state", s.baseURL, connectionId)
	req := api.ChangeConnectionLifecycleStateRequest{
		State: state,
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	var connection api.Connection
	if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodPost, url, ctx.ToHeaders(), payload, &connection); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return &connection, nil
}

// api/v1/credential [get]
func (s *onboardClient) ListCredentials(ctx *httpclient.Context, connector []source.Type, credentialType *api.CredentialType, health *string, pageSize, pageNumber int) (api.ListCredentialResponse, error) {
	url := fmt.Sprintf("%s/api/v1/credential", s.baseURL)

	firstParamAttached := false
	if len(connector) > 0 {
		for _, v := range connector {
			if !firstParamAttached {
				url += "?"
				firstParamAttached = true
			} else {
				url += "&"
			}
			url += "connector=" + string(v)
		}
	}
	if credentialType != nil && *credentialType != "" {
		if !firstParamAttached {
			url += "?"
			firstParamAttached = true
		} else {
			url += "&"
		}
		url += "credentialType=" + string(*credentialType)
	}
	if health != nil && *health != "" {
		if !firstParamAttached {
			url += "?"
			firstParamAttached = true
		} else {
			url += "&"
		}
		url += "health=" + string(*health)
	}
	if pageSize > 0 {
		if !firstParamAttached {
			url += "?"
			firstParamAttached = true
		} else {
			url += "&"
		}
		url += fmt.Sprintf("pageSize=%d", pageSize)
	}
	if pageNumber > 0 {
		if !firstParamAttached {
			url += "?"
			firstParamAttached = true
		} else {
			url += "&"
		}
		url += fmt.Sprintf("pageNumber=%d", pageNumber)
	}

	var response api.ListCredentialResponse
	if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return api.ListCredentialResponse{}, echo.NewHTTPError(statusCode, err.Error())
		}
		return api.ListCredentialResponse{}, err
	}
	return response, nil
}

func (s *onboardClient) TriggerAutoOnboard(ctx *httpclient.Context, credentialId string) ([]api.Connection, error) {
	url := fmt.Sprintf("%s/api/v1/credential/%s/autoonboard", s.baseURL, credentialId)

	var response []api.Connection
	if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodPost, url, ctx.ToHeaders(), nil, &response); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}

	return response, nil
}

func (s *onboardClient) GetConnectionGroup(ctx *httpclient.Context, connectionGroupName string) (*api.ConnectionGroup, error) {
	url := fmt.Sprintf("%s/api/v1/connection-groups/%s", s.baseURL, connectionGroupName)

	var connectionGroup api.ConnectionGroup
	if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodGet, url, ctx.ToHeaders(), nil, &connectionGroup); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}

	return &connectionGroup, nil
}

func (s *onboardClient) ListConnectionGroups(ctx *httpclient.Context) ([]api.ConnectionGroup, error) {
	url := fmt.Sprintf("%s/api/v1/connection-groups", s.baseURL)

	var connectionGroup []api.ConnectionGroup
	if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodGet, url, ctx.ToHeaders(), nil, &connectionGroup); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}

	return connectionGroup, nil
}
