package client

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/httprequest"
)

type BenchmarkAssignment struct {
	BenchmarkId string `json:"benchmarkId"`
	SourceId    string `json:"sourceId"`
	AssignedAt  int64  `json:"assignedAt"`
}

func GetBenchmarkAssignmentsBySourceId(baseUrl string, sourceID uuid.UUID) ([]BenchmarkAssignment, error) {
	url := fmt.Sprintf("%s/api/v1/benchmarks/source/%s", baseUrl, sourceID.String())

	assignments := []BenchmarkAssignment{}
	if err := httprequest.DoRequest(http.MethodGet, url, nil, nil, &assignments); err != nil {
		return nil, err
	}
	return assignments, nil
}
