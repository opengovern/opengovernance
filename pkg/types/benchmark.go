package types

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type FullBenchmark struct {
	ID    string `json:"ID" example:"azure_cis_v140"` // Benchmark ID
	Title string `json:"title" example:"CIS v1.4.0"`  // Benchmark title
}

type Finding struct {
	ID               string           `json:"ID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1-azure_cis_v140_7_5"` // Finding ID
	BenchmarkID      string           `json:"benchmarkID" example:"azure_cis_v140"`                                                                                    // Benchmark ID
	PolicyID         string           `json:"policyID" example:"azure_cis_v140_7_5"`                                                                                   // Policy ID
	ConnectionID     string           `json:"connectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`                                                             // Connection ID
	DescribedAt      int64            `json:"describedAt" example:"1589395200"`                                                                                        // Timestamp of the policy description
	EvaluatedAt      int64            `json:"evaluatedAt" example:"1589395200"`                                                                                        // Timestamp of the policy evaluation
	StateActive      bool             `json:"stateActive" example:"true"`                                                                                              // Whether the policy is active or not
	Result           ComplianceResult `json:"result" example:"alarm"`                                                                                                  // Compliance result
	Severity         Severity         `json:"severity" example:"low"`                                                                                                  // Compliance severity
	Evaluator        string           `json:"evaluator" example:"steampipe-v0.5"`                                                                                      // Evaluator name
	Connector        source.Type      `json:"connector" example:"Azure"`                                                                                               // Cloud provider
	ResourceID       string           `json:"resourceID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"`            // Resource ID
	ResourceName     string           `json:"resourceName" example:"vm-1"`                                                                                             // Resource name
	ResourceLocation string           `json:"resourceLocation" example:"eastus"`                                                                                       // Resource location
	ResourceType     string           `json:"resourceType" example:"Microsoft.Compute/virtualMachines"`                                                                // Resource type
	Reason           string           `json:"reason" example:"The VM is not using managed disks"`                                                                      // Reason for the policy evaluation result
	ComplianceJobID  uint             `json:"complianceJobID" example:"1"`                                                                                             // Compliance job ID
	ScheduleJobID    uint             `json:"scheduleJobID" example:"1"`                                                                                               // Schedule job ID
}

func (r Finding) KeysAndIndex() ([]string, string) {
	index := FindingsIndex
	if strings.HasPrefix(r.ConnectionID, "stack-") {
		index = StackFindingsIndex
	}
	return []string{
		r.ResourceID,
		r.ConnectionID,
		r.PolicyID,
		strconv.FormatInt(r.DescribedAt, 10),
	}, index
}

type BenchmarkReportType string

const (
	BenchmarksSummary                 BenchmarkReportType = "BenchmarksSummary"
	BenchmarksSummaryHistory          BenchmarkReportType = "BenchmarksSummaryHistory"
	BenchmarksConnectorSummary        BenchmarkReportType = "BenchmarksConnectorSummary"
	BenchmarksConnectorSummaryHistory BenchmarkReportType = "BenchmarksConnectorHistory"
)

type ResourceResult struct {
	ResourceID   string           `json:"resource_id"`
	ResourceName string           `json:"resource_name"`
	ConnectionID string           `json:"connection_id"`
	Result       ComplianceResult `json:"result"`
}

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

	TotalResult   ComplianceResultSummary `json:"total_result"`
	TotalSeverity SeverityResult          `json:"total_severity"`

	ReportType BenchmarkReportType `json:"report_type"`

	SummarizeJobId uint `json:"summarize_job_id"`
}

func (r BenchmarkSummary) KeysAndIndex() ([]string, string) {
	connectionsTypesStr, _ := json.Marshal(r.ConnectorTypes)
	keys := []string{
		r.BenchmarkID,
		r.ConnectionID,
		string(connectionsTypesStr),
		string(BenchmarksSummary),
	}
	if r.ReportType == BenchmarksSummaryHistory {
		keys = append(keys, fmt.Sprintf("%d", r.DescribedAt))
	}
	return keys, BenchmarkSummaryIndex
}
