package client

import (
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	"github.com/labstack/echo/v4"
	"net/http"
	"time"

	compliance "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type ComplianceServiceClient interface {
	ListAssignmentsByBenchmark(ctx *httpclient.Context, benchmarkID string) (*compliance.BenchmarkAssignedEntities, error)
	GetBenchmark(ctx *httpclient.Context, benchmarkID string) (*compliance.Benchmark, error)
	GetControl(ctx *httpclient.Context, controlID string) (*compliance.Control, error)
	GetQuery(ctx *httpclient.Context, queryID string) (*compliance.Query, error)
	ListInsightsMetadata(ctx *httpclient.Context, connectors []source.Type) ([]compliance.Insight, error)
	GetFindings(ctx *httpclient.Context, req compliance.GetFindingsRequest) (compliance.GetFindingsResponse, error)
	GetInsight(ctx *httpclient.Context, insightId string, connectionId []string, startTime *time.Time, endTime *time.Time) (compliance.Insight, error)
	ListBenchmarks(ctx *httpclient.Context) ([]compliance.Benchmark, error)
	ListAllBenchmarks(ctx *httpclient.Context) ([]compliance.Benchmark, error)
	GetAccountsFindingsSummary(ctx *httpclient.Context, benchmarkId string, connectionId []string, connector []source.Type) (compliance.GetAccountsFindingsSummaryResponse, error)
	ListInsights(ctx *httpclient.Context) ([]compliance.Insight, error)
	CreateBenchmarkAssignment(ctx *httpclient.Context, benchmarkID, connectionId string) ([]compliance.BenchmarkAssignment, error)
	CountFindings(ctx *httpclient.Context, conformanceStatuses []compliance.ConformanceStatus) (*compliance.CountFindingsResponse, error)
	ListQueries(ctx *httpclient.Context) ([]compliance.Query, error)
	ListControl(ctx *httpclient.Context) ([]compliance.Control, error)
}

type complianceClient struct {
	baseURL string
}

func NewComplianceClient(baseURL string) ComplianceServiceClient {
	return &complianceClient{baseURL: baseURL}
}

func (s *complianceClient) ListAssignmentsByBenchmark(ctx *httpclient.Context, benchmarkID string) (*compliance.BenchmarkAssignedEntities, error) {
	url := fmt.Sprintf("%s/api/v1/assignments/benchmark/%s", s.baseURL, benchmarkID)

	var response compliance.BenchmarkAssignedEntities
	if statusCode, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return &response, nil
}

func (s *complianceClient) GetBenchmark(ctx *httpclient.Context, benchmarkID string) (*compliance.Benchmark, error) {
	url := fmt.Sprintf("%s/api/v1/benchmarks/%s", s.baseURL, benchmarkID)

	var response compliance.Benchmark
	if statusCode, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return &response, nil
}

func (s *complianceClient) GetControl(ctx *httpclient.Context, controlID string) (*compliance.Control, error) {
	url := fmt.Sprintf("%s/api/v1/benchmarks/controls/%s", s.baseURL, controlID)

	var response compliance.Control
	if statusCode, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return &response, nil
}

func (s *complianceClient) ListControl(ctx *httpclient.Context) ([]compliance.Control, error) {
	url := fmt.Sprintf("%s/api/v1/benchmarks/controls", s.baseURL)

	var response []compliance.Control
	if statusCode, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return response, nil
}

func (s *complianceClient) ListQueries(ctx *httpclient.Context) ([]compliance.Query, error) {
	url := fmt.Sprintf("%s/api/v1/benchmarks/queries", s.baseURL)

	var response []compliance.Query
	if statusCode, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return response, nil
}

func (s *complianceClient) GetQuery(ctx *httpclient.Context, queryID string) (*compliance.Query, error) {
	url := fmt.Sprintf("%s/api/v1/queries/%s", s.baseURL, queryID)

	var response compliance.Query
	if statusCode, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return &response, nil
}

func (s *complianceClient) ListInsightsMetadata(ctx *httpclient.Context, connectors []source.Type) ([]compliance.Insight, error) {
	url := fmt.Sprintf("%s/api/v1/metadata/insight", s.baseURL)
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

	var insights []compliance.Insight
	if statusCode, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &insights); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return insights, nil
}

func (s *complianceClient) GetFindings(ctx *httpclient.Context, req compliance.GetFindingsRequest) (compliance.GetFindingsResponse, error) {
	url := fmt.Sprintf("%s/api/v1/findings", s.baseURL)

	payload, err := json.Marshal(req)
	if err != nil {
		return compliance.GetFindingsResponse{}, err
	}

	var response compliance.GetFindingsResponse
	if statusCode, err := httpclient.DoRequest(http.MethodPost, url, ctx.ToHeaders(), payload, &response); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return compliance.GetFindingsResponse{}, echo.NewHTTPError(statusCode, err.Error())
		}
		return compliance.GetFindingsResponse{}, err
	}

	return response, nil
}

func (s *complianceClient) CountFindings(ctx *httpclient.Context, conformanceStatuses []compliance.ConformanceStatus) (*compliance.CountFindingsResponse, error) {
	url := fmt.Sprintf("%s/api/v1/findings/count", s.baseURL)

	if len(conformanceStatuses) == 0 {
		conformanceStatuses = compliance.ListConformanceStatuses()
	}

	isFirstParamAttached := false
	for _, conformanceStatus := range conformanceStatuses {
		if !isFirstParamAttached {
			url += "?"
			isFirstParamAttached = true
		} else {
			url += "&"
		}
		url += fmt.Sprintf("conformanceStatus=%s", conformanceStatus)
	}

	var response compliance.CountFindingsResponse
	if statusCode, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}

	return &response, nil
}

func (s *complianceClient) GetInsight(ctx *httpclient.Context, insightId string, connectionIDs []string, startTime *time.Time, endTime *time.Time) (compliance.Insight, error) {
	url := fmt.Sprintf("%s/api/v1/insight/%s", s.baseURL, insightId)
	firstParamAttached := false
	if len(connectionIDs) > 0 {
		for _, connectionID := range connectionIDs {
			if !firstParamAttached {
				url += "?"
				firstParamAttached = true
			} else {
				url += "&"
			}
			url += fmt.Sprintf("connectionId=%s", connectionID)
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

	var insight compliance.Insight
	if statusCode, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &insight); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return compliance.Insight{}, echo.NewHTTPError(statusCode, err.Error())
		}
		return compliance.Insight{}, err
	}
	return insight, nil
}

func (s *complianceClient) ListBenchmarks(ctx *httpclient.Context) ([]compliance.Benchmark, error) {
	url := fmt.Sprintf("%s/api/v1/benchmarks", s.baseURL)

	var benchmarks []compliance.Benchmark
	if statusCode, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &benchmarks); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return benchmarks, nil
}

func (s *complianceClient) ListAllBenchmarks(ctx *httpclient.Context) ([]compliance.Benchmark, error) {
	url := fmt.Sprintf("%s/api/v1/benchmarks/all", s.baseURL)

	var benchmarks []compliance.Benchmark
	if statusCode, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &benchmarks); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return benchmarks, nil
}

func (s *complianceClient) GetAccountsFindingsSummary(ctx *httpclient.Context, benchmarkId string, connectionIds []string, connector []source.Type) (compliance.GetAccountsFindingsSummaryResponse, error) {
	url := fmt.Sprintf("%s/api/v1/findings/%s/accounts", s.baseURL, benchmarkId)

	var firstParamAttached bool
	firstParamAttached = false

	if len(connectionIds) > 0 {
		for _, connectionId := range connectionIds {
			if !firstParamAttached {
				url += "?"
				firstParamAttached = true
			} else {
				url += "&"
			}
			url += fmt.Sprintf("connectionId=%v", &connectionId)
		}
	}

	if connector != nil {
		if !firstParamAttached {
			url += "?"
			firstParamAttached = true
		} else {
			url += "&"
		}
		url += fmt.Sprintf("connector=%v", &connector)
	}

	var res compliance.GetAccountsFindingsSummaryResponse
	if statusCode, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &res); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return compliance.GetAccountsFindingsSummaryResponse{}, echo.NewHTTPError(statusCode, err.Error())
		}
		return compliance.GetAccountsFindingsSummaryResponse{}, err
	}
	return res, nil
}

func (s *complianceClient) ListInsights(ctx *httpclient.Context) ([]compliance.Insight, error) {
	url := fmt.Sprintf("%s/api/v1/insight", s.baseURL)

	var insights []compliance.Insight
	if statusCode, err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &insights); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return insights, nil
}

func (s *complianceClient) CreateBenchmarkAssignment(ctx *httpclient.Context, benchmarkID, connectionId string) ([]compliance.BenchmarkAssignment, error) {
	url := fmt.Sprintf("%s/api/v1/assignments/%s/connection?connectionId=%s", s.baseURL, benchmarkID, connectionId)

	var assignments []compliance.BenchmarkAssignment
	if statusCode, err := httpclient.DoRequest(http.MethodPost, url, ctx.ToHeaders(), nil, &assignments); err != nil {
		if 400 <= statusCode && statusCode < 500 {
			return nil, echo.NewHTTPError(statusCode, err.Error())
		}
		return nil, err
	}
	return assignments, nil
}
