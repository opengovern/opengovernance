package client

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	insight "gitlab.com/keibiengine/keibi-engine/pkg/insight/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"

	"github.com/kaytu-io/kaytu-util/pkg/source"
	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/api"
)

type InventoryServiceClient interface {
	CountResources(ctx *httpclient.Context) (int64, error)
	GetAccountsResourceCount(ctx *httpclient.Context, provider source.Type, sourceId *string) ([]api.ConnectionResourceCountResponse, error)
	ListInsightResults(ctx *httpclient.Context, connectors []source.Type, connectionIds []string, insightIds []uint, timeAt *time.Time) (map[uint][]insight.InsightResource, error)
	GetInsightResult(ctx *httpclient.Context, connectionIds []string, insightId uint, timeAt *time.Time) ([]insight.InsightResource, error)
	GetInsightTrendResults(ctx *httpclient.Context, connectionIds []string, insightId uint, timeStart, timeEnd *time.Time) (map[int][]insight.InsightResource, error)
	ListConnectionsData(ctx *httpclient.Context, connectionIds []string, timeStart, timeEnd *time.Time) (map[string]api.ConnectionData, error)
	GetConnectionData(ctx *httpclient.Context, connectionId string, timeStart, timeEnd *time.Time) (*api.ConnectionData, error)
}

type inventoryClient struct {
	baseURL string
}

func NewInventoryServiceClient(baseURL string) InventoryServiceClient {
	return &inventoryClient{baseURL: baseURL}
}

func (s *inventoryClient) CountResources(ctx *httpclient.Context) (int64, error) {
	url := fmt.Sprintf("%s/api/v1/resources/count", s.baseURL)

	var count int64
	if statusCode, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &count); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return 0, echo.NewHTTPError(statusCode, err.Error())
		}
		return 0, err
	}
	return count, nil
}

func (s *inventoryClient) GetAccountsResourceCount(ctx *httpclient.Context, provider source.Type, sourceId *string) ([]api.ConnectionResourceCountResponse, error) {
	url := fmt.Sprintf("%s/api/v1/accounts/resource/count?provider=%s", s.baseURL, provider.String())
	if sourceId != nil {
		url += "&sourceId=" + *sourceId
	}

	var response []api.ConnectionResourceCountResponse
	if statusCode, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return response, nil
}

func (s *inventoryClient) ListInsightResults(ctx *httpclient.Context, connectors []source.Type, connectionIds []string, insightIds []uint, timeAt *time.Time) (map[uint][]insight.InsightResource, error) {
	url := fmt.Sprintf("%s/api/v2/insights", s.baseURL)
	firstParamAttached := false
	if len(connectors) > 0 {
		for _, connector := range connectors {
			if !firstParamAttached {
				url += "?"
				firstParamAttached = true
			} else {
				url += "&"
			}
			url += fmt.Sprintf("connector=%s", connector.String())
		}
	}
	if len(connectionIds) > 0 {
		for _, connectionId := range connectionIds {
			if !firstParamAttached {
				url += "?"
				firstParamAttached = true
			} else {
				url += "&"
			}
			url += fmt.Sprintf("connectionId=%s", connectionId)
		}
	}
	if len(insightIds) > 0 {
		for _, insightId := range insightIds {
			if !firstParamAttached {
				url += "?"
				firstParamAttached = true
			} else {
				url += "&"
			}
			url += fmt.Sprintf("insightId=%d", insightId)
		}
	}
	if timeAt != nil {
		if !firstParamAttached {
			url += "?"
			firstParamAttached = true
		} else {
			url += "&"
		}
		url += fmt.Sprintf("time=%d", timeAt.Unix())
	}

	var response map[uint][]insight.InsightResource
	if _, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		return nil, err
	}
	return response, nil
}

func (s *inventoryClient) GetInsightResult(ctx *httpclient.Context, connectionIds []string, insightId uint, timeAt *time.Time) ([]insight.InsightResource, error) {
	url := fmt.Sprintf("%s/api/v2/insights/%d", s.baseURL, insightId)
	firstParamAttached := false
	if len(connectionIds) > 0 {
		for _, connectionId := range connectionIds {
			if !firstParamAttached {
				url += "?"
				firstParamAttached = true
			} else {
				url += "&"
			}
			url += fmt.Sprintf("connectionId=%s", connectionId)
		}
	}
	if timeAt != nil {
		if !firstParamAttached {
			url += "?"
			firstParamAttached = true
		} else {
			url += "&"
		}
		url += fmt.Sprintf("time=%d", timeAt.Unix())
	}

	var response []insight.InsightResource
	if statusCode, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return response, nil
}

func (s *inventoryClient) GetInsightTrendResults(ctx *httpclient.Context, connectionIds []string, insightId uint, timeStart, timeEnd *time.Time) (map[int][]insight.InsightResource, error) {
	url := fmt.Sprintf("%s/api/v2/insights/%d/trend", s.baseURL, insightId)
	firstParamAttached := false
	if len(connectionIds) > 0 {
		for _, connectionId := range connectionIds {
			if !firstParamAttached {
				url += "?"
				firstParamAttached = true
			} else {
				url += "&"
			}
			url += fmt.Sprintf("connectionId=%s", connectionId)
		}
	}
	if timeStart != nil {
		if !firstParamAttached {
			url += "?"
			firstParamAttached = true
		} else {
			url += "&"
		}
		url += fmt.Sprintf("timeStart=%d", timeStart.Unix())
	}
	if timeEnd != nil {
		if !firstParamAttached {
			url += "?"
			firstParamAttached = true
		} else {
			url += "&"
		}
		url += fmt.Sprintf("timeEnd=%d", timeEnd.Unix())
	}

	var response map[int][]insight.InsightResource
	if statusCode, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return response, nil
}

func (s *inventoryClient) ListConnectionsData(ctx *httpclient.Context, connectionIds []string, timeStart, timeEnd *time.Time) (map[string]api.ConnectionData, error) {
	url := fmt.Sprintf("%s/api/v2/connections/data", s.baseURL)
	firstParamAttached := false
	if len(connectionIds) > 0 {
		for _, connectionId := range connectionIds {
			if !firstParamAttached {
				url += "?"
				firstParamAttached = true
			} else {
				url += "&"
			}
			url += fmt.Sprintf("connectionId=%s", connectionId)
		}
	}
	if timeStart != nil {
		if !firstParamAttached {
			url += "?"
			firstParamAttached = true
		} else {
			url += "&"
		}
		url += fmt.Sprintf("startTime=%d", timeStart.Unix())
	}
	if timeEnd != nil {
		if !firstParamAttached {
			url += "?"
			firstParamAttached = true
		} else {
			url += "&"
		}
		url += fmt.Sprintf("endTime=%d", timeEnd.Unix())
	}

	var response map[string]api.ConnectionData
	if statusCode, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return response, nil
}

func (s *inventoryClient) GetConnectionData(ctx *httpclient.Context, connectionId string, timeStart, timeEnd *time.Time) (*api.ConnectionData, error) {
	url := fmt.Sprintf("%s/api/v2/connections/data/%s", s.baseURL, connectionId)
	firstParamAttached := false
	if timeStart != nil {
		if !firstParamAttached {
			url += "?"
			firstParamAttached = true
		} else {
			url += "&"
		}
		url += fmt.Sprintf("startTime=%d", timeStart.Unix())
	}
	if timeEnd != nil {
		if !firstParamAttached {
			url += "?"
			firstParamAttached = true
		} else {
			url += "&"
		}
		url += fmt.Sprintf("endTime=%d", timeEnd.Unix())
	}

	var response api.ConnectionData
	if statusCode, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return &response, nil
}
