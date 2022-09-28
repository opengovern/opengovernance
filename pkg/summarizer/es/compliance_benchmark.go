package es

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/es"
)

const (
	BenchmarkSummaryIndex = "benchmark_summary"
)

type BenchmarkReportType string

const (
	BenchmarksSummary BenchmarkReportType = "TrendPerSourceHistory"
)

type ResourceResult struct {
	ResourceID string    `json:"resource_id"`
	SourceID   uuid.UUID `json:"source_id"`
	Result     es.Status `json:"result"`
}

type PolicySummary struct {
	PolicyID  string           `json:"policy_id"`
	Resources []ResourceResult `json:"resources"`
}

type BenchmarkSummary struct {
	BenchmarkID   string          `json:"benchmark_id"`
	ScheduleJobID uint            `json:"schedule_job_id"`
	DescribedAt   int64           `json:"described_at"`
	EvaluatedAt   int64           `json:"evaluated_at"`
	Policies      []PolicySummary `json:"policies"`

	ReportType BenchmarkReportType `json:"report_type"`
}

func (r BenchmarkSummary) KeysAndIndex() ([]string, string) {
	keys := []string{
		r.BenchmarkID,
		string(BenchmarksSummary),
	}
	if strings.HasSuffix(string(r.ReportType), "History") {
		keys = append(keys, fmt.Sprintf("%d", r.ScheduleJobID))
	}
	return keys, BenchmarkSummaryIndex
}
