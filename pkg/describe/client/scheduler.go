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
	GetConnectionDescribeStatus(ctx *httpclient.Context, connectionID string) ([]api.ConnectionDescribeStatus, error)
	ListPendingConnections(ctx *httpclient.Context) ([]string, error)
	GetLatestComplianceJobForBenchmark(ctx *httpclient.Context, benchmarkID string) (*api.ComplianceJob, error)
	GetDescribeAllJobsStatus(ctx *httpclient.Context) (*api.DescribeAllJobsStatus, error)
	TriggerAnalyticsJob(ctx *httpclient.Context) error
	TriggerInsightJob(ctx *httpclient.Context, insightID uint) error
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

func (s *schedulerClient) GetDescribeAllJobsStatus(ctx *httpclient.Context) (*api.DescribeAllJobsStatus, error) {
	url := fmt.Sprintf("%s/describe/all/jobs/state", s.baseURL)

	var status api.DescribeAllJobsStatus
	if statusCode, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &status); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return &status, nil
}

func (s *schedulerClient) TriggerAnalyticsJob(ctx *httpclient.Context) error {
	url := fmt.Sprintf("%s/analytics/trigger", s.baseURL)

	if statusCode, err := httpclient.DoRequest(http.MethodPut, url, ctx.ToHeaders(), nil, nil); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return echo.NewHTTPError(statusCode, err.Error())
		}
		return err
	}
	return nil
}

func (s *schedulerClient) TriggerInsightJob(ctx *httpclient.Context, insightID uint) error {
	url := fmt.Sprintf("%s/insight/trigger/%d", s.baseURL, insightID)

	if statusCode, err := httpclient.DoRequest(http.MethodPut, url, ctx.ToHeaders(), nil, nil); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return echo.NewHTTPError(statusCode, err.Error())
		}
		return err
	}
	return nil
}

func (s *schedulerClient) GetDescribeStatus(ctx *httpclient.Context, resourceType string) ([]api.DescribeStatus, error) {
	url := fmt.Sprintf("%s/api/v1/describe/status/%s", s.baseURL, resourceType)

	var res []api.DescribeStatus
	if statusCode, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &res); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return res, nil
}

func (s *schedulerClient) GetLatestComplianceJobForBenchmark(ctx *httpclient.Context, benchmarkId string) (*api.ComplianceJob, error) {
	url := fmt.Sprintf("%s/api/v1/compliance/status/%s", s.baseURL, benchmarkId)

	var res api.ComplianceJob
	if statusCode, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &res); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return &res, nil
}

func (s *schedulerClient) GetConnectionDescribeStatus(ctx *httpclient.Context, connectionID string) ([]api.ConnectionDescribeStatus, error) {
	url := fmt.Sprintf("%s/api/v1/describe/connection/status?connection_id=%s", s.baseURL, connectionID)

	var res []api.ConnectionDescribeStatus
	if statusCode, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &res); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return res, nil
}

func (s *schedulerClient) ListPendingConnections(ctx *httpclient.Context) ([]string, error) {
	url := fmt.Sprintf("%s/api/v1/describe/pending/connections", s.baseURL)

	var res []string
	if statusCode, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &res); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return res, nil
}
