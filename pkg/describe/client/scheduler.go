package client

import (
	"fmt"
	"net/http"

	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"github.com/labstack/echo/v4"

	compliance "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/api"
)

type TimeRangeFilter struct {
	From int64 // from epoch millisecond
	To   int64 // from epoch millisecond
}
type SchedulerServiceClient interface {
	GetSource(ctx *httpclient.Context, sourceID string) (*api.Source, error)
	ListComplianceReportJobs(ctx *httpclient.Context, sourceID string, filter *TimeRangeFilter) ([]*compliance.ComplianceReport, error)
	GetLastComplianceReportID(ctx *httpclient.Context) (uint, error)
	GetInsightJobById(ctx *httpclient.Context, jobId string) (api.InsightJob, error)
	GetStack(ctx *httpclient.Context, stackID string) (*api.Stack, error)
}

type schedulerClient struct {
	baseURL string
}

func NewSchedulerServiceClient(baseURL string) SchedulerServiceClient {
	return &schedulerClient{baseURL: baseURL}
}

func (s *schedulerClient) GetSource(ctx *httpclient.Context, sourceID string) (*api.Source, error) {
	url := fmt.Sprintf("%s/api/v1/sources/%s", s.baseURL, sourceID)

	var source api.Source
	if statusCode, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &source); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return &source, nil
}

func (s *schedulerClient) ListComplianceReportJobs(ctx *httpclient.Context, sourceID string, filter *TimeRangeFilter) ([]*compliance.ComplianceReport, error) {
	url := fmt.Sprintf("%s/api/v1/sources/%s/jobs/compliance", s.baseURL, sourceID)
	if filter != nil {
		url = fmt.Sprintf("%s?from=%d&to=%d", url, filter.From, filter.To)
	}

	reports := []*compliance.ComplianceReport{}
	if statusCode, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &reports); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return reports, nil
}

func (s *schedulerClient) GetLastComplianceReportID(ctx *httpclient.Context) (uint, error) {
	url := fmt.Sprintf("%s/api/v1/compliance/report/last/completed", s.baseURL)

	var id uint
	if statusCode, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &id); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return 0, echo.NewHTTPError(statusCode, err.Error())
		}
		return 0, err
	}
	return id, nil
}

func (s *schedulerClient) GetInsightJobById(ctx *httpclient.Context, jobId string) (api.InsightJob, error) {
	url := fmt.Sprintf("%s/api/v1/insight/job/%s", s.baseURL, jobId)

	var job api.InsightJob
	if _, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &job); err != nil {
		return api.InsightJob{}, err
	}
	return job, nil
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
