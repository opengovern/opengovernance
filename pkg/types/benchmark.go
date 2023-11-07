package types

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type FullBenchmark struct {
	ID    string `json:"ID" example:"azure_cis_v140"` // Benchmark ID
	Title string `json:"title" example:"CIS v1.4.0"`  // Benchmark title
}

type Finding struct {
	BenchmarkID      string           `json:"benchmarkID" example:"azure_cis_v140"`
	PolicyID         string           `json:"policyID" example:"azure_cis_v140_7_5"`
	ConnectionID     string           `json:"connectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`
	EvaluatedAt      int64            `json:"evaluatedAt" example:"1589395200"`
	StateActive      bool             `json:"stateActive" example:"true"`
	Result           ComplianceResult `json:"result" example:"alarm"`
	Severity         FindingSeverity  `json:"severity" example:"low"`
	Evaluator        string           `json:"evaluator" example:"steampipe-v0.5"`
	Connector        source.Type      `json:"connector" example:"Azure"`
	ResourceID       string           `json:"resourceID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"`
	ResourceName     string           `json:"resourceName" example:"vm-1"`
	ResourceLocation string           `json:"resourceLocation" example:"eastus"`
	ResourceType     string           `json:"resourceType" example:"Microsoft.Compute/virtualMachines"`
	Reason           string           `json:"reason" example:"The VM is not using managed disks"`
	ComplianceJobID  uint             `json:"complianceJobID" example:"1"`

	ResourceCollection *string  `json:"resourceCollection"` // Resource collection
	ParentBenchmarks   []string `json:"parentBenchmarks"`
}

func (r Finding) KeysAndIndex() ([]string, string) {
	index := FindingsIndex
	keys := []string{
		r.ResourceID,
		r.ConnectionID,
		r.PolicyID,
	}
	if r.ResourceCollection != nil {
		keys = append(keys, *r.ResourceCollection)
		index = ResourceCollectionsFindingsIndex
	}
	if strings.HasPrefix(r.ConnectionID, "stack-") {
		index = StackFindingsIndex
	}
	return keys, index
}

type BenchmarkReportType string

const (
	BenchmarksSummary                 BenchmarkReportType = "BenchmarksSummary"
	BenchmarksSummaryHistory          BenchmarkReportType = "BenchmarksSummaryHistory"
	BenchmarksConnectorSummary        BenchmarkReportType = "BenchmarksConnectorSummary"
	BenchmarksConnectorSummaryHistory BenchmarkReportType = "BenchmarksConnectorHistory"
)

type PolicySummary struct {
	PolicyID      string                  `json:"policy_id"`
	ConnectorType source.Type             `json:"connector_type"`
	TotalResult   ComplianceResultSummary `json:"total_result"`
	TotalSeverity SeverityResult          `json:"total_severity"`
}

type BenchmarkSummary struct {
	BenchmarkID    string          `json:"benchmark_id"`
	ConnectionID   string          `json:"connection_id"`
	ConnectorTypes []source.Type   `json:"connector_types"`
	DescribedAt    int64           `json:"described_at"`
	EvaluatedAt    int64           `json:"evaluated_at"`
	Policies       []PolicySummary `json:"policies"`

	FailedResources map[string]struct{}
	AllResources    map[string]struct{}
	Resources       ComplianceResultShortSummary

	TotalResult   ComplianceResultSummary `json:"total_result"`
	TotalSeverity SeverityResult          `json:"total_severity"`

	ReportType BenchmarkReportType `json:"report_type"`

	SummarizeJobId uint `json:"summarize_job_id"`

	ResourceCollection *string `json:"resource_collection"`
}

func (r BenchmarkSummary) KeysAndIndex() ([]string, string) {
	connectionsTypesStr, _ := json.Marshal(r.ConnectorTypes)
	keys := []string{
		r.BenchmarkID,
		r.ConnectionID,
		string(connectionsTypesStr),
		string(r.ReportType),
	}
	if r.ReportType == BenchmarksSummaryHistory ||
		r.ReportType == BenchmarksConnectorSummaryHistory {
		keys = append(keys, fmt.Sprintf("%d", r.DescribedAt))
	}
	idx := BenchmarkSummaryIndex
	if r.ResourceCollection != nil {
		keys = append(keys, *r.ResourceCollection)
		idx = ResourceCollectionsBenchmarkSummaryIndex
	}
	return keys, idx
}
