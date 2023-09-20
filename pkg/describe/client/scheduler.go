package client

import (
	"fmt"
	"net/http"

	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"github.com/labstack/echo/v4"

	"github.com/kaytu-io/kaytu-engine/pkg/describe/api"
)

type TimeRangeFilter struct {
	From int64 // from epoch millisecond
	To   int64 // from epoch millisecond
}
type SchedulerServiceClient interface {
	GetStack(ctx *httpclient.Context, stackID string) (*api.Stack, error)
	GetDescribeStatus(ctx *httpclient.Context, resourceType string) ([]api.DescribeStatus, error)
}

type schedulerClient struct {
	baseURL string
}

func NewSchedulerServiceClient(baseURL string) SchedulerServiceClient {
	return &schedulerClient{baseURL: baseURL}
}

func (s *schedulerClient) GetStack(ctx *httpclient.Context, stackID string) (*api.Stack, error) {
	url := fmt.Sprintf("%s/api/v1/stacks/%s", s.baseURL, stackID)

	var stack api.Stack
	if statusCode, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &stack); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return &stack, nil
}

func (s *schedulerClient) GetDescribeStatus(ctx *httpclient.Context, resourceType string) ([]api.DescribeStatus, error) {
	url := fmt.Sprintf("%s/api/v1/describe/status?resource_type=%s", s.baseURL, resourceType)

	var res []api.DescribeStatus
	if statusCode, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &res); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return res, nil
}
