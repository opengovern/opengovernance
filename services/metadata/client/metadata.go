package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/opengovern/og-util/pkg/httpclient"
	"github.com/opengovern/opencomply/services/metadata/api"

	"github.com/opengovern/opencomply/services/metadata/models"
)

var ErrConfigNotFound = errors.New("config not found")

type MetadataServiceClient interface {
	GetConfigMetadata(ctx *httpclient.Context, key models.MetadataKey) (models.IConfigMetadata, error)
	SetConfigMetadata(ctx *httpclient.Context, key models.MetadataKey, value any) error
	ListQueryParameters(ctx *httpclient.Context) (api.ListQueryParametersResponse, error)
	SetQueryParameter(ctx *httpclient.Context, request api.SetQueryParameterRequest) error
	VaultConfigured(ctx *httpclient.Context) (*string, error)
	GetViewsCheckpoint(ctx *httpclient.Context) (*api.GetViewsCheckpointResponse, error)
	ReloadViews(ctx *httpclient.Context) error
	GetAbout(ctx *httpclient.Context) (*api.About, error)
}

type metadataClient struct {
	baseURL string
}

func NewMetadataServiceClient(baseURL string) MetadataServiceClient {
	return &metadataClient{
		baseURL: baseURL,
	}
}

func (s *metadataClient) GetConfigMetadata(ctx *httpclient.Context, key models.MetadataKey) (models.IConfigMetadata, error) {
	url := fmt.Sprintf("%s/api/v1/metadata/%s", s.baseURL, string(key))
	var cnf models.ConfigMetadata
	if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodGet, url, ctx.ToHeaders(), nil, &cnf); err != nil {
		if statusCode == 404 {
			return nil, ErrConfigNotFound
		}
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}

	switch cnf.Type {
	case models.ConfigMetadataTypeString:
		return &models.StringConfigMetadata{
			ConfigMetadata: cnf,
		}, nil
	case models.ConfigMetadataTypeInt:
		intValue, err := strconv.ParseInt(cnf.Value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to convert string to int: %w", err)
		}
		return &models.IntConfigMetadata{
			ConfigMetadata: cnf,
			Value:          int(intValue),
		}, nil
	case models.ConfigMetadataTypeBool:
		boolValue, err := strconv.ParseBool(cnf.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to convert string to bool: %w", err)
		}
		return &models.BoolConfigMetadata{
			ConfigMetadata: cnf,
			Value:          boolValue,
		}, nil
	case models.ConfigMetadataTypeJSON:
		return &models.JSONConfigMetadata{
			ConfigMetadata: cnf,
			Value:          cnf.Value,
		}, nil
	}

	return nil, fmt.Errorf("unknown config metadata type: %s", cnf.Type)
}

func (s *metadataClient) SetConfigMetadata(ctx *httpclient.Context, key models.MetadataKey, value any) error {
	url := fmt.Sprintf("%s/api/v1/metadata", s.baseURL)

	req := api.SetConfigMetadataRequest{
		Key:   string(key),
		Value: value,
	}
	jsonReq, err := json.Marshal(req)
	if err != nil {
		return err
	}

	var cnf models.ConfigMetadata
	if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodPost, url, ctx.ToHeaders(), jsonReq, &cnf); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return echo.NewHTTPError(statusCode, err.Error())
		}
		return err
	}

	return nil
}

func (s *metadataClient) ListQueryParameters(ctx *httpclient.Context) (api.ListQueryParametersResponse, error) {
	url := fmt.Sprintf("%s/api/v1/query_parameter", s.baseURL)
	var resp api.ListQueryParametersResponse
	if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodGet, url, ctx.ToHeaders(), nil, &resp); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return resp, echo.NewHTTPError(statusCode, err.Error())
		}
		return resp, err
	}
	return resp, nil
}

func (s *metadataClient) SetQueryParameter(ctx *httpclient.Context, request api.SetQueryParameterRequest) error {
	url := fmt.Sprintf("%s/api/v1/query_parameter", s.baseURL)
	jsonReq, err := json.Marshal(request)
	if err != nil {
		return err
	}

	if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodPost, url, ctx.ToHeaders(), jsonReq, nil); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return echo.NewHTTPError(statusCode, err.Error())
		}
		return err
	}

	return nil
}

func (s *metadataClient) VaultConfigured(ctx *httpclient.Context) (*string, error) {
	url := fmt.Sprintf("%s/api/v3/vault/configured", s.baseURL)
	var status string
	if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodGet, url, ctx.ToHeaders(), nil, &status); err != nil {
		if statusCode == 404 {
			return nil, ErrConfigNotFound
		}
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}

	return &status, nil
}

func (s *metadataClient) ReloadViews(ctx *httpclient.Context) error {
	url := fmt.Sprintf("%s/api/v1/views/reload", s.baseURL)

	if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodPut, url, ctx.ToHeaders(), nil, nil); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return echo.NewHTTPError(statusCode, err.Error())
		}
		return err
	}
	return nil
}

func (s *metadataClient) GetViewsCheckpoint(ctx *httpclient.Context) (*api.GetViewsCheckpointResponse, error) {
	url := fmt.Sprintf("%s/api/v1/views/checkpoint", s.baseURL)
	var resp api.GetViewsCheckpointResponse
	if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodGet, url, ctx.ToHeaders(), nil, &resp); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return &resp, nil
}

func (s *metadataClient) GetAbout(ctx *httpclient.Context) (*api.About, error) {
	url := fmt.Sprintf("%s/api/v3/about", s.baseURL)

	var about api.About
	if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodGet, url, ctx.ToHeaders(), nil, &about); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}

	return &about, nil
}
