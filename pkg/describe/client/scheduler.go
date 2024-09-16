package client

import (
	"fmt"
	"github.com/kaytu-io/kaytu-util/pkg/httpclient"
	"github.com/kaytu-io/open-governance/pkg/describe/db/model"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/kaytu-io/open-governance/pkg/describe/api"
)

type TimeRangeFilter struct {
	From int64 // from epoch millisecond
	To   int64 // from epoch millisecond
}
type SchedulerServiceClient interface {
	GetDescribeStatus(ctx *httpclient.Context, resourceType string) ([]api.DescribeStatus, error)
	GetConnectionDescribeStatus(ctx *httpclient.Context, connectionID string) ([]api.ConnectionDescribeStatus, error)
	ListPendingConnections(ctx *httpclient.Context) ([]string, error)
	GetLatestComplianceJobForBenchmark(ctx *httpclient.Context, benchmarkID string) (*api.ComplianceJob, error)
	GetDescribeAllJobsStatus(ctx *httpclient.Context) (*api.DescribeAllJobsStatus, error)
	TriggerAnalyticsJob(ctx *httpclient.Context) (uint, error)
	GetAnalyticsJob(ctx *httpclient.Context, jobID uint) (*model.AnalyticsJob, error)
	CountJobsByDate(ctx *httpclient.Context, includeCost *bool, jobType api.JobType, startDate, endDate time.Time) (int64, error)
}

type schedulerClient struct {
	baseURL string
}

func NewSchedulerServiceClient(baseURL string) SchedulerServiceClient {
	return &schedulerClient{baseURL: baseURL}
}

func (s *schedulerClient) GetDescribeAllJobsStatus(ctx *httpclient.Context) (*api.DescribeAllJobsStatus, error) {
	url := fmt.Sprintf("%s/api/v1/describe/all/jobs/state", s.baseURL)

	var status api.DescribeAllJobsStatus
	if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodGet, url, ctx.ToHeaders(), nil, &status); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return &status, nil
}

func (s *schedulerClient) TriggerAnalyticsJob(ctx *httpclient.Context) (uint, error) {
	url := fmt.Sprintf("%s/api/v1/analytics/trigger", s.baseURL)

	var jobID uint
	if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodPut, url, ctx.ToHeaders(), nil, &jobID); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return 0, echo.NewHTTPError(statusCode, err.Error())
		}
		return 0, err
	}
	return jobID, nil
}

func (s *schedulerClient) GetDescribeStatus(ctx *httpclient.Context, resourceType string) ([]api.DescribeStatus, error) {
	url := fmt.Sprintf("%s/api/v1/describe/status/%s", s.baseURL, resourceType)

	var res []api.DescribeStatus
	if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodGet, url, ctx.ToHeaders(), nil, &res); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return res, nil
}

func (s *schedulerClient) GetLatestComplianceJobForBenchmark(ctx *httpclient.Context, benchmarkId string) (*api.ComplianceJob, error) {
	url := fmt.Sprintf("%s/api/v1/compliance/status/%s", s.baseURL, benchmarkId)

	var res *api.ComplianceJob
	if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodGet, url, ctx.ToHeaders(), nil, &res); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return res, nil
}

func (s *schedulerClient) GetAnalyticsJob(ctx *httpclient.Context, jobID uint) (*model.AnalyticsJob, error) {
	url := fmt.Sprintf("%s/api/v1/analytics/job/%d", s.baseURL, jobID)

	var res *model.AnalyticsJob
	if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodGet, url, ctx.ToHeaders(), nil, &res); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return res, nil
}

func (s *schedulerClient) GetConnectionDescribeStatus(ctx *httpclient.Context, connectionID string) ([]api.ConnectionDescribeStatus, error) {
	url := fmt.Sprintf("%s/api/v1/describe/connection/status?connection_id=%s", s.baseURL, connectionID)

	var res []api.ConnectionDescribeStatus
	if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodGet, url, ctx.ToHeaders(), nil, &res); err != nil {
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
	if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodGet, url, ctx.ToHeaders(), nil, &res); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return res, nil
}

func (s *schedulerClient) CountJobsByDate(ctx *httpclient.Context, includeCost *bool, jobType api.JobType, startDate, endDate time.Time) (int64, error) {
	url := fmt.Sprintf("%s/api/v1/jobs/bydate?startDate=%d&endDate=%d&jobType=%s", s.baseURL, startDate.UnixMilli(), endDate.UnixMilli(), string(jobType))
	if includeCost != nil {
		url += fmt.Sprintf("&include_cost=%v", *includeCost)
	}

	var resp int64
	if statusCode, err := httpclient.DoRequest(ctx.Ctx, http.MethodGet, url, ctx.ToHeaders(), nil, &resp); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return 0, echo.NewHTTPError(statusCode, err.Error())
		}
		return 0, err
	}
	return resp, nil
}
