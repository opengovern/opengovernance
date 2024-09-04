package api

import (
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"time"

	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type Benchmark struct {
	ID                string              `json:"id" example:"azure_cis_v140"`                                                                                                                                                       // Benchmark ID
	Title             string              `json:"title" example:"Azure CIS v1.4.0"`                                                                                                                                                  // Benchmark title
	ReferenceCode     string              `json:"referenceCode" example:"CIS 1.4.0"`                                                                                                                                                 // Benchmark display code
	Description       string              `json:"description" example:"The CIS Microsoft Azure Foundations Security Benchmark provides prescriptive guidance for establishing a secure baseline configuration for Microsoft Azure."` // Benchmark description
	LogoURI           string              `json:"logoURI"`                                                                                                                                                                           // Benchmark logo URI
	Category          string              `json:"category"`                                                                                                                                                                          // Benchmark category
	DocumentURI       string              `json:"documentURI" example:"benchmarks/azure_cis_v140.md"`                                                                                                                                // Benchmark document URI
	AutoAssign        bool                `json:"autoAssign" example:"true"`                                                                                                                                                         // Whether the benchmark is auto assigned or not
	TracksDriftEvents bool                `json:"tracksDriftEvents" example:"true"`                                                                                                                                                  // Whether the benchmark tracks drift events or not
	Tags              map[string][]string `json:"tags" `                                                                                                                                                                             // Benchmark tags
	Connectors        []source.Type       `json:"connectors" example:"[azure]"`                                                                                                                                                      // Benchmark connectors
	Children          []string            `json:"children" example:"[azure_cis_v140_1, azure_cis_v140_2]"`                                                                                                                           // Benchmark children
	Controls          []string            `json:"controls" example:"[azure_cis_v140_1_1, azure_cis_v140_1_2]"`                                                                                                                       // Benchmark controls
	CreatedAt         time.Time           `json:"createdAt" example:"2020-01-01T00:00:00Z"`                                                                                                                                          // Benchmark creation date
	UpdatedAt         time.Time           `json:"updatedAt" example:"2020-01-01T00:00:00Z"`                                                                                                                                          // Benchmark last update date
}

type BenchmarkTrendDatapoint struct {
	Timestamp                time.Time                       `json:"timestamp" example:"1686346668"`
	ConformanceStatusSummary ConformanceStatusSummary        `json:"conformanceStatusSummary"`
	Checks                   types.SeverityResult            `json:"checks"`
	ControlsSeverityStatus   BenchmarkControlsSeverityStatus `json:"controlsSeverityStatus"`
}

type ListBenchmarksSummaryResponse struct {
	BenchmarkSummary []BenchmarkEvaluationSummary `json:"benchmarkSummary"`

	TotalConformanceStatusSummary ConformanceStatusSummary `json:"totalConformanceStatusSummary"`
	TotalChecks                   types.SeverityResult     `json:"totalChecks"`
}

type BenchmarkStatusResult struct {
	PassedCount int `json:"passed"`
	TotalCount  int `json:"total"`
}

type BenchmarkControlsSeverityStatus struct {
	Total BenchmarkStatusResult `json:"total"`

	Critical BenchmarkStatusResult `json:"critical"`
	High     BenchmarkStatusResult `json:"high"`
	Medium   BenchmarkStatusResult `json:"medium"`
	Low      BenchmarkStatusResult `json:"low"`
	None     BenchmarkStatusResult `json:"none"`
}

type BenchmarkResourcesSeverityStatus struct {
	Total BenchmarkStatusResult `json:"total"`

	Critical BenchmarkStatusResult `json:"critical"`
	High     BenchmarkStatusResult `json:"high"`
	Medium   BenchmarkStatusResult `json:"medium"`
	Low      BenchmarkStatusResult `json:"low"`
	None     BenchmarkStatusResult `json:"none"`
}

type ConformanceStatusSummary struct {
	PassedCount int `json:"passed"`
	FailedCount int `json:"failed"`
}

func (c *ConformanceStatusSummary) AddESConformanceStatusMap(summary map[types.ConformanceStatus]int) {
	c.PassedCount += summary[types.ConformanceStatusOK]
	c.FailedCount += summary[types.ConformanceStatusALARM]
	c.PassedCount += summary[types.ConformanceStatusINFO]
	c.PassedCount += summary[types.ConformanceStatusSKIP]
	c.FailedCount += summary[types.ConformanceStatusERROR]
}

type BenchmarkEvaluationSummary struct {
	Benchmark
	ConformanceStatusSummary ConformanceStatusSummary         `json:"conformanceStatusSummary"`
	Checks                   types.SeverityResult             `json:"checks"`
	ControlsSeverityStatus   BenchmarkControlsSeverityStatus  `json:"controlsSeverityStatus"`
	ResourcesSeverityStatus  BenchmarkResourcesSeverityStatus `json:"resourcesSeverityStatus"`
	ConnectionsStatus        BenchmarkStatusResult            `json:"connectionsStatus"`
	CostOptimization         *float64                         `json:"costOptimization"`
	EvaluatedAt              *time.Time                       `json:"evaluatedAt" example:"2020-01-01T00:00:00Z"`
	LastJobStatus            string                           `json:"lastJobStatus" example:"success"`
	TopConnections           []TopFieldRecord                 `json:"topConnections"`
}

type BenchmarkControlSummary struct {
	Benchmark Benchmark                 `json:"benchmark"`
	Controls  []ControlSummary          `json:"control"`
	Children  []BenchmarkControlSummary `json:"children"`
}
