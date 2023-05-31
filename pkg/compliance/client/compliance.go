package client

import (
	"encoding/json"
	"fmt"
	"net/http"

	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"

	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	compliance "gitlab.com/keibiengine/keibi-engine/pkg/compliance/api"
)

type ComplianceServiceClient interface {
	GetAllBenchmarkAssignmentsBySourceId(ctx *httpclient.Context, sourceID uuid.UUID) ([]compliance.BenchmarkAssignment, error)
	GetBenchmark(ctx *httpclient.Context, benchmarkID string) (*compliance.Benchmark, error)
	GetPolicy(ctx *httpclient.Context, policyID string) (*compliance.Policy, error)
	GetQuery(ctx *httpclient.Context, queryID string) (*compliance.Query, error)
	GetInsights(ctx *httpclient.Context, connector source.Type) ([]compliance.Insight, error)
	GetInsightById(ctx *httpclient.Context, id uint) (*compliance.Insight, error)
	GetInsightPeerGroups(ctx *httpclient.Context, connector source.Type) ([]compliance.InsightPeerGroup, error)
	GetInsightPeerGroupById(ctx *httpclient.Context, id uint) (*compliance.InsightPeerGroup, error)
	GetFindings(ctx *httpclient.Context, sourceIDs []string, benchmarkID string, resourceIDs []string) (compliance.GetFindingsResponse, error)
}

type complianceClient struct {
	baseURL string
}

func NewComplianceClient(baseURL string) ComplianceServiceClient {
	return &complianceClient{baseURL: baseURL}
}

func (s *complianceClient) GetAllBenchmarkAssignmentsBySourceId(ctx *httpclient.Context, sourceID uuid.UUID) ([]compliance.BenchmarkAssignment, error) {
	url := fmt.Sprintf("%s/api/v1/assignments/connection/%s", s.baseURL, sourceID.String())

	var response []compliance.BenchmarkAssignment
	if err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		return nil, err
	}
	return response, nil
}

func (s *complianceClient) GetBenchmark(ctx *httpclient.Context, benchmarkID string) (*compliance.Benchmark, error) {
	url := fmt.Sprintf("%s/api/v1/benchmarks/%s", s.baseURL, benchmarkID)

	var response compliance.Benchmark
	if err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (s *complianceClient) GetPolicy(ctx *httpclient.Context, policyID string) (*compliance.Policy, error) {
	url := fmt.Sprintf("%s/api/v1/benchmarks/policies/%s", s.baseURL, policyID)

	var response compliance.Policy
	if err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (s *complianceClient) GetQuery(ctx *httpclient.Context, queryID string) (*compliance.Query, error) {
	url := fmt.Sprintf("%s/api/v1/queries/%s", s.baseURL, queryID)

	var response compliance.Query
	if err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func (s *complianceClient) GetInsights(ctx *httpclient.Context, connector source.Type) ([]compliance.Insight, error) {
	url := fmt.Sprintf("%s/api/v1/insight", s.baseURL)
	if connector != source.Nil {
		url = fmt.Sprintf("%s?connector=%s", url, connector)
	}

	var insights []compliance.Insight
	if err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &insights); err != nil {
		return nil, err
	}
	return insights, nil
}

func (s *complianceClient) GetInsightById(ctx *httpclient.Context, id uint) (*compliance.Insight, error) {
	url := fmt.Sprintf("%s/api/v1/insight/%d", s.baseURL, id)

	var insight compliance.Insight
	if err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &insight); err != nil {
		return nil, err
	}
	return &insight, nil
}

func (s *complianceClient) GetInsightPeerGroups(ctx *httpclient.Context, connector source.Type) ([]compliance.InsightPeerGroup, error) {
	url := fmt.Sprintf("%s/api/v1/insight/peer", s.baseURL)
	if connector != source.Nil {
		url = fmt.Sprintf("%s?connector=%s", url, connector)
	}

	var insightPeerGroups []compliance.InsightPeerGroup
	if err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &insightPeerGroups); err != nil {
		return nil, err
	}
	return insightPeerGroups, nil
}

func (s *complianceClient) GetInsightPeerGroupById(ctx *httpclient.Context, id uint) (*compliance.InsightPeerGroup, error) {
	url := fmt.Sprintf("%s/api/v1/insight/peer/%d", s.baseURL, id)

	var insightPeerGroup compliance.InsightPeerGroup
	if err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &insightPeerGroup); err != nil {
		return nil, err
	}
	return &insightPeerGroup, nil
}

func (s *complianceClient) GetFindings(ctx *httpclient.Context, sourceIDs []string, benchmarkID string, resourceIDs []string) (compliance.GetFindingsResponse, error) {
	url := fmt.Sprintf("%s/api/v1/findings", s.baseURL)

	req := compliance.GetFindingsRequest{
		Filters: compliance.FindingFilters{
			ConnectionID: sourceIDs,
			BenchmarkID:  []string{benchmarkID},
			ResourceID:   resourceIDs,
		},
		Sorts: []compliance.FindingSortItem{
			{
				Field:     "status",
				Direction: "desc",
			},
		},
		Page: compliance.Page{
			No:   1,
			Size: 100,
		},
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return compliance.GetFindingsResponse{}, err
	}

	var response compliance.GetFindingsResponse
	if err := httpclient.DoRequest(http.MethodPost, url, ctx.ToHeaders(), payload, &response); err != nil {
		return compliance.GetFindingsResponse{}, err
	}

	return response, nil
}
