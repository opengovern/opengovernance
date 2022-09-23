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
}

type complianceClient struct {
	baseURL string
}

func NewComplianceClient(baseURL string) ComplianceServiceClient {
	return &complianceClient{baseURL: baseURL}
}

func (s *complianceClient) GetAllBenchmarkAssignmentsBySourceId(ctx *httpclient.Context, sourceID uuid.UUID) ([]compliance.BenchmarkAssignment, error) {
	url := fmt.Sprintf("%s/api/v1/benchmarks/source/%s", s.baseURL, sourceID.String())

	var response []compliance.BenchmarkAssignment
	if err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &response); err != nil {
		return nil, err
	}
	return response, nil
}
