package api

import (
	"time"

	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type Benchmark struct {
	ID          string              `json:"id" example:"azure_cis_v140"`                                                                                                                                                       // Benchmark ID
	Title       string              `json:"title" example:"Azure CIS v1.4.0"`                                                                                                                                                  // Benchmark title
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
	Policies    []string            `json:"policies" example:"[azure_cis_v140_1_1, azure_cis_v140_1_2]"`                                                                                                                       // Benchmark policies
	CreatedAt   time.Time           `json:"createdAt" example:"2020-01-01T00:00:00Z"`                                                                                                                                          // Benchmark creation date
	UpdatedAt   time.Time           `json:"updatedAt" example:"2020-01-01T00:00:00Z"`                                                                                                                                          // Benchmark last update date
}

type Policy struct {
	ID                 string              `json:"id" example:"azure_cis_v140_1_1"`
	Title              string              `json:"title" example:"1.1 Ensure that multi-factor authentication status is enabled for all privileged users"`
	Description        string              `json:"description" example:"Enable multi-factor authentication for all user credentials who have write access to Azure resources. These include roles like 'Service Co-Administrators', 'Subscription Owners', 'Contributors'."`
	Tags               map[string][]string `json:"tags" `
	Connector          source.Type         `json:"connector" example:"Azure"`
	Enabled            bool                `json:"enabled" example:"true"`
	DocumentURI        string              `json:"documentURI" example:"benchmarks/azure_cis_v140_1_1.md"`
	QueryID            *string             `json:"queryID" example:"azure_ad_manual_control"`
	Severity           types.Severity      `json:"severity" example:"low"`
	ManualVerification bool                `json:"manualVerification" example:"true"`
	Managed            bool                `json:"managed" example:"true"`
	CreatedAt          time.Time           `json:"createdAt" example:"2020-01-01T00:00:00Z"`
	UpdatedAt          time.Time           `json:"updatedAt" example:"2020-01-01T00:00:00Z"`
}

type Query struct {
	ID             string    `json:"id" example:"azure_ad_manual_control"`
	QueryToExecute string    `json:"queryToExecute" example:"select\n  -- Required Columns\n  'active_directory' as resource,\n  'info' as status,\n  'Manual verification required.' as reason;\n"`
	Connector      string    `json:"connector" example:"Azure"`
	ListOfTables   []string  `json:"listOfTables" example:"null"`
	Engine         string    `json:"engine" example:"steampipe-v0.5"`
	CreatedAt      time.Time `json:"createdAt" example:"2023-06-07T14:00:15.677558Z"`
	UpdatedAt      time.Time `json:"updatedAt" example:"2023-06-16T14:58:08.759554Z"`
}

type BenchmarkTrendDatapoint struct {
	Timestamp int                           `json:"timestamp" example:"1686346668"` // Time
	Result    types.ComplianceResultSummary `json:"result"`
	Checks    types.SeverityResult          `json:"checks"`
}
