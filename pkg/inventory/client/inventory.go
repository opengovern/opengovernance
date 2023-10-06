package client

import (
	"fmt"
	"net/http"
	url2 "net/url"
	"strconv"
	"time"

	insight "github.com/kaytu-io/kaytu-engine/pkg/insight/es"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"github.com/labstack/echo/v4"

	"github.com/kaytu-io/kaytu-engine/pkg/inventory/api"
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type InventoryServiceClient interface {
	CountResources(ctx *httpclient.Context) (int64, error)
	ListInsightResults(ctx *httpclient.Context, connectors []source.Type, connectionIds []string, insightIds []uint, timeAt *time.Time) (map[uint][]insight.InsightResource, error)
	GetInsightResult(ctx *httpclient.Context, connectionIds []string, insightId uint, timeAt *time.Time) ([]insight.InsightResource, error)
	GetInsightTrendResults(ctx *httpclient.Context, connectionIds []string, insightId uint, startTime, endTime *time.Time) (map[int][]insight.InsightResource, error)
	ListConnectionsData(ctx *httpclient.Context, connectionIds []string, startTime, endTime *time.Time, needCost, needResourceCount bool) (map[string]api.ConnectionData, error)
	ListResourceTypesMetadata(ctx *httpclient.Context, connectors []source.Type, services []string, resourceTypes []string, summarized bool, tags map[string]string, pageSize, pageNumber int) (*api.ListResourceTypeMetadataResponse, error)
	GetResourceCollection(ctx *httpclient.Context, id string) (*api.ResourceCollection, error)
}

type inventoryClient struct {
	baseURL string
}

func NewInventoryServiceClient(baseURL string) InventoryServiceClient {
	return &inventoryClient{baseURL: baseURL}
}

func (s *inventoryClient) CountResources(ctx *httpclient.Context) (int64, error) {
	url := fmt.Sprintf("%s/api/v2/resources/count", s.baseURL)

	var count int64
	if statusCode, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &count); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return 0, echo.NewHTTPError(statusCode, err.Error())
		}
		return 0, err
	}
	return count, nil
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

func (s *inventoryClient) GetInsightTrendResults(ctx *httpclient.Context, connectionIds []string, insightId uint, startTime, endTime *time.Time) (map[int][]insight.InsightResource, error) {
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
	if startTime != nil {
		if !firstParamAttached {
			url += "?"
			firstParamAttached = true
		} else {
			url += "&"
		}
		url += fmt.Sprintf("startTime=%d", startTime.Unix())
	}
	if endTime != nil {
		if !firstParamAttached {
			url += "?"
			firstParamAttached = true
		} else {
			url += "&"
		}
		url += fmt.Sprintf("endTime=%d", endTime.Unix())
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

func (s *inventoryClient) ListConnectionsData(ctx *httpclient.Context, connectionIds []string, startTime, endTime *time.Time, needCost, needResourceCount bool) (map[string]api.ConnectionData, error) {
	url := fmt.Sprintf("%s/api/v2/connections/data", s.baseURL)
	params := url2.Values{}
	if len(connectionIds) > 0 {
		for _, connectionId := range connectionIds {
			params.Set("connectionId", connectionId)
		}
	}
	if startTime != nil {
		params.Set("startTime", strconv.FormatInt(startTime.Unix(), 10))
	}
	if endTime != nil {
		params.Set("endTime", strconv.FormatInt(endTime.Unix(), 10))
	}
	if !needCost {
		params.Set("needCost", "false")
	}
	if !needResourceCount {
		params.Set("needResourceCount", "false")
	}
	if len(params) > 0 {
		url += "?" + params.Encode()
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

func (s *inventoryClient) ListResourceTypesMetadata(ctx *httpclient.Context, connectors []source.Type, services []string, resourceTypes []string, summarized bool, tags map[string]string, pageSize, pageNumber int) (*api.ListResourceTypeMetadataResponse, error) {
	url := fmt.Sprintf("%s/api/v2/metadata/resourcetype", s.baseURL)
	firstParamAttached := false
	if len(connectors) > 0 {
		for _, connector := range connectors {
			if !firstParamAttached {
				url += "?"
				firstParamAttached = true
			} else {
				url += "&"
			}
			url += fmt.Sprintf("connector=%s", connector)
		}
	}
	if len(services) > 0 {
		for _, service := range services {
			if !firstParamAttached {
				url += "?"
				firstParamAttached = true
			} else {
				url += "&"
			}
			url += fmt.Sprintf("service=%s", service)
		}
	}
	if len(resourceTypes) > 0 {
		for _, resourceType := range resourceTypes {
			if !firstParamAttached {
				url += "?"
				firstParamAttached = true
			} else {
				url += "&"
			}
			url += fmt.Sprintf("resourceType=%s", resourceType)
		}
	}
	if summarized {
		if !firstParamAttached {
			url += "?"
			firstParamAttached = true
		} else {
			url += "&"
		}
		url += "summarized=true"
	}
	if len(tags) > 0 {
		for key, value := range tags {
			if !firstParamAttached {
				url += "?"
				firstParamAttached = true
			} else {
				url += "&"
			}
			url += fmt.Sprintf("tags=%s=%s", key, value)
		}
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

	var response api.ListResourceTypeMetadataResponse
	if statusCode, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return &response, nil
}

func (s *inventoryClient) GetResourceCollection(ctx *httpclient.Context, id string) (*api.ResourceCollection, error) {
	url := fmt.Sprintf("%s/api/v2/resource-collection/%s", s.baseURL, id)

	var response api.ResourceCollection
	if statusCode, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return &response, nil
}
