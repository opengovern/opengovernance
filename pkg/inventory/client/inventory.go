package client

import (
	"fmt"
	"net/http"

	"gitlab.com/keibiengine/keibi-engine/pkg/internal/httpclient"
)

type BenchmarkAssignment struct {
	BenchmarkId string `json:"benchmarkId"`
	SourceId    string `json:"sourceId"`
	AssignedAt  int64  `json:"assignedAt"`
}

type InventoryServiceClient interface {
	GetAllBenchmarkAssignmentsBySourceId(ctx *httpclient.Context, sourceID string) ([]*BenchmarkAssignment, error)
	CountResources(ctx *httpclient.Context) (int64, error)
}

type inventoryClient struct {
	baseURL string
}

func NewInventoryServiceClient(baseURL string) InventoryServiceClient {
	return &inventoryClient{baseURL: baseURL}
}

func (s *inventoryClient) GetAllBenchmarkAssignmentsBySourceId(ctx *httpclient.Context, sourceID string) ([]*BenchmarkAssignment, error) {
	url := fmt.Sprintf("%s/api/v1/benchmarks/source/%s", s.baseURL, sourceID)

	assignments := []*BenchmarkAssignment{}
	if err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &assignments); err != nil {
		return nil, err
	}
	return assignments, nil
}

func (s *inventoryClient) CountResources(ctx *httpclient.Context) (int64, error) {
	url := fmt.Sprintf("%s/api/v1/resources/count", s.baseURL)

	var count int64
	if err := httpclient.DoRequest(http.MethodGet, url, ctx.ToHeaders(), nil, &count); err != nil {
		return 0, err
	}
	return count, nil
}
