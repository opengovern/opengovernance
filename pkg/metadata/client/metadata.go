package client

import (
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/metadata/api"
	"github.com/labstack/echo/v4"
	"net/http"
	"strconv"

	"github.com/kaytu-io/kaytu-engine/pkg/metadata/models"
)

type MetadataServiceClient interface {
	GetConfigMetadata(ctx *httpclient.Context, key models.MetadataKey) (models.IConfigMetadata, error)
	SetConfigMetadata(ctx *httpclient.Context, key models.MetadataKey, value any) error
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
	if statusCode, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &cnf); err != nil {
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
	if statusCode, err := httpclient.DoRequest(http.MethodPost, url, ctx.ToHeaders(), jsonReq, &cnf); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return echo.NewHTTPError(statusCode, err.Error())
		}
		return err
	}

	return nil
}
