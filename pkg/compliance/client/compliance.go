package client

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	compliance "gitlab.com/keibiengine/keibi-engine/pkg/compliance/api"

	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
)

type ComplianceServiceClient interface {
	GetAllBenchmarkAssignmentsBySourceId(ctx *httpclient.Context, sourceID uuid.UUID) ([]compliance.BenchmarkAssignment, error)
	GetBenchmark(ctx *httpclient.Context, benchmarkID string) (*compliance.Benchmark, error)
	GetPolicy(ctx *httpclient.Context, policyID string) (*compliance.Policy, error)
	GetQuery(ctx *httpclient.Context, queryID string) (*compliance.Query, error)
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
