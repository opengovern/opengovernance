package api

import (
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"time"

	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type Benchmark struct {
	ID          string              `json:"id" example:"azure_cis_v140"`                                                                                                                                                       // Benchmark ID
	Title       string              `json:"title" example:"Azure CIS v1.4.0"`                                                                                                                                                  // Benchmark title
	DisplayCode string              `json:"displayCode" example:"CIS 1.4.0"`                                                                                                                                                   // Benchmark display code
	Description string              `json:"description" example:"The CIS Microsoft Azure Foundations Security Benchmark provides prescriptive guidance for establishing a secure baseline configuration for Microsoft Azure."` // Benchmark description
	LogoURI     string              `json:"logoURI"`                                                                                                                                                                           // Benchmark logo URI
	Category    string              `json:"category"`                                                                                                                                                                          // Benchmark category
	DocumentURI string              `json:"documentURI" example:"benchmarks/azure_cis_v140.md"`                                                                                                                                // Benchmark document URI
	Enabled     bool                `json:"enabled" example:"true"`                                                                                                                                                            // Whether the benchmark is enabled or not
	Managed     bool                `json:"managed" example:"true"`                                                                                                                                                            // Whether the benchmark is managed or not
	AutoAssign  bool                `json:"autoAssign" example:"true"`                                                                                                                                                         // Whether the benchmark is auto assigned or not
	Baseline    bool                `json:"baseline" example:"true"`                                                                                                                                                           // Whether the benchmark is baseline or not
	Tags        map[string][]string `json:"tags" `                                                                                                                                                                             // Benchmark tags
	Connectors  []source.Type       `json:"connectors" example:"[azure]"`                                                                                                                                                      // Benchmark connectors
	Children    []string            `json:"children" example:"[azure_cis_v140_1, azure_cis_v140_2]"`                                                                                                                           // Benchmark children
	Controls    []string            `json:"controls" example:"[azure_cis_v140_1_1, azure_cis_v140_1_2]"`                                                                                                                       // Benchmark controls
	CreatedAt   time.Time           `json:"createdAt" example:"2020-01-01T00:00:00Z"`                                                                                                                                          // Benchmark creation date
	UpdatedAt   time.Time           `json:"updatedAt" example:"2020-01-01T00:00:00Z"`                                                                                                                                          // Benchmark last update date
}

type BenchmarkTrendDatapoint struct {
	Timestamp     int     `json:"timestamp" example:"1686346668"` // Time
	SecurityScore float64 `json:"securityScore"`
}

type GetBenchmarksSummaryResponse struct {
	BenchmarkSummary []BenchmarkEvaluationSummary `json:"benchmarkSummary"`

	TotalConformanceStatusSummary types.ConformanceStatusSummary `json:"totalConformanceStatusSummary"`
	TotalChecks                   types.SeverityResult           `json:"totalChecks"`
}

type BenchmarkControlsSeverityStatusResult struct {
	PassedCount int `json:"passed"`
	TotalCount  int `json:"total"`
}

type BenchmarkControlsSeverityStatus struct {
	Total BenchmarkControlsSeverityStatusResult `json:"total"`

	Critical BenchmarkControlsSeverityStatusResult `json:"critical"`
	High     BenchmarkControlsSeverityStatusResult `json:"high"`
	Medium   BenchmarkControlsSeverityStatusResult `json:"medium"`
	Low      BenchmarkControlsSeverityStatusResult `json:"low"`
	None     BenchmarkControlsSeverityStatusResult `json:"none"`
}

type BenchmarkEvaluationSummary struct {
	Benchmark
	ConformanceStatusSummary types.ConformanceStatusSummary  `json:"conformanceStatusSummary"`                   // Compliance result summary
	Checks                   types.SeverityResult            `json:"checks"`                                     // Checks summary
	ControlsSeverityStatus   BenchmarkControlsSeverityStatus `json:"controlsSeverityStatus"`                     // Controls severity status
	EvaluatedAt              *time.Time                      `json:"evaluatedAt" example:"2020-01-01T00:00:00Z"` // Evaluated at
	LastJobStatus            string                          `json:"lastJobStatus" example:"success"`            // Last job status
	TopConnections           []TopFieldRecord                `json:"topConnections"`                             // Top connections
}

type BenchmarkControlSummary struct {
	Benchmark Benchmark                 `json:"benchmark"`
	Controls  []ControlSummary          `json:"control"`
	Children  []BenchmarkControlSummary `json:"children"`
}
