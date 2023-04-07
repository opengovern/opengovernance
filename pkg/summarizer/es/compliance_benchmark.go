package es

import (
	"fmt"
	"strings"

	"gitlab.com/keibiengine/keibi-engine/pkg/types"
)

const (
	BenchmarkSummaryIndex = "benchmark_summary"
)

type BenchmarkReportType string

const (
	BenchmarksSummary        BenchmarkReportType = "BenchmarksSummary"
	BenchmarksSummaryHistory BenchmarkReportType = "BenchmarksSummaryHistory"
)

type ResourceResult struct {
	ResourceID   string                 `json:"resource_id"`
	ResourceName string                 `json:"resource_name"`
	SourceID     string                 `json:"source_id"`
	Result       types.ComplianceResult `json:"result"`
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
