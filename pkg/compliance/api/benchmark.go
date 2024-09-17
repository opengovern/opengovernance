package api

import (
	"github.com/kaytu-io/open-governance/pkg/types"
	"strings"
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

type BenchmarkStatusResultV2 struct {
	TotalCount  int `json:"total"`
	PassedCount int `json:"passed"`
	FailedCount int `json:"failed"`
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

type BenchmarkControlsSeverityStatusV2 struct {
	Total BenchmarkStatusResultV2 `json:"total"`

	Critical BenchmarkStatusResultV2 `json:"critical"`
	High     BenchmarkStatusResultV2 `json:"high"`
	Medium   BenchmarkStatusResultV2 `json:"medium"`
	Low      BenchmarkStatusResultV2 `json:"low"`
	None     BenchmarkStatusResultV2 `json:"none"`
}

type BenchmarkResourcesSeverityStatusV2 struct {
	Total BenchmarkStatusResultV2 `json:"total"`

	Critical BenchmarkStatusResultV2 `json:"critical"`
	High     BenchmarkStatusResultV2 `json:"high"`
	Medium   BenchmarkStatusResultV2 `json:"medium"`
	Low      BenchmarkStatusResultV2 `json:"low"`
	None     BenchmarkStatusResultV2 `json:"none"`
}

type ConformanceStatusSummary struct {
	PassedCount int `json:"passed"`
	FailedCount int `json:"failed"`
}

type ConformanceStatusSummaryV2 struct {
	TotalCount  int `json:"total_count"`
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

func (c *ConformanceStatusSummaryV2) AddESConformanceStatusMap(summary map[types.ConformanceStatus]int) {
	c.PassedCount += summary[types.ConformanceStatusOK]
	c.FailedCount += summary[types.ConformanceStatusALARM]
	c.PassedCount += summary[types.ConformanceStatusINFO]
	c.PassedCount += summary[types.ConformanceStatusSKIP]
	c.FailedCount += summary[types.ConformanceStatusERROR]

	c.TotalCount = c.FailedCount + c.PassedCount
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

type GetBenchmarkDetailsRequest struct {
	TagsRegex         *string                `json:"tags_regex"`
	FindingFilters    *FindingSummaryFilters `json:"finding_filters"`
	BenchmarkChildren bool                   `json:"benchmark_children"`
}

type GetBenchmarkDetailsMetadata struct {
	ID                string              `json:"id"`
	Title             string              `json:"title"`
	Description       string              `json:"description"`
	Enabled           bool                `json:"enabled"`
	TrackDriftEvents  bool                `json:"track_drift_events"`
	Connectors        []source.Type       `json:"connectors"`
	NumberOfControls  int                 `json:"number_of_controls"`
	SupportedControls []string            `json:"supported_controls"`
	PrimaryTables     []string            `json:"primary_tables"`
	ListOfTables      []string            `json:"list_of_tables"`
	Tags              map[string][]string `json:"tags"`
	CreatedAt         time.Time           `json:"created_at"`
	UpdatedAt         time.Time           `json:"updated_at"`
}

type GetBenchmarkDetailsFindings struct {
	Results         map[types.ConformanceStatus]int `json:"results"`
	LastEvaluatedAt time.Time                       `json:"lastEvaluated_at"`
	ConnectionIDs   []string                        `json:"connection_ids"`
}

type GetBenchmarkDetailsChildren struct {
	ID         string                        `json:"id"`
	Title      string                        `json:"title"`
	Tags       map[string][]string           `json:"tags"`
	ControlIDs []string                      `json:"control_ids"`
	Findings   GetBenchmarkDetailsFindings   `json:"findings"`
	Children   []GetBenchmarkDetailsChildren `json:"children"`
}

type GetBenchmarkDetailsResponse struct {
	Metadata GetBenchmarkDetailsMetadata   `json:"metadata"`
	Findings GetBenchmarkDetailsFindings   `json:"findings"`
	Children []GetBenchmarkDetailsChildren `json:"children"`
}

type GetBenchmarkListRequest struct {
	ParentBenchmarkID []string               `json:"parent_benchmark_id"`
	Tags              map[string][]string    `json:"tags"`
	TagsRegex         *string                `json:"tags_regex"`
	PrimaryTable      []string               `json:"primary_table"`
	ListOfTables      []string               `json:"list_of_tables"`
	Root              bool                   `json:"root"`
	FindingFilters    *FindingSummaryFilters `json:"finding_filters"`
	FindingSummary    bool                   `json:"finding_summary"`
	Cursor            *int64                 `json:"cursor"`
	PerPage           *int64                 `json:"per_page"`
}

type GetBenchmarkListMetadata struct {
	ID               string              `json:"id"`
	Title            string              `json:"title"`
	Description      string              `json:"description"`
	Connectors       []source.Type       `json:"connectors"`
	NumberOfControls int                 `json:"number_of_controls"`
	Enabled          bool                `json:"enabled"`
	TrackDriftEvents bool                `json:"track_drift_events"`
	PrimaryTables    []string            `json:"primary_tables"`
	Tags             map[string][]string `json:"tags"`
	CreatedAt        time.Time           `json:"created_at"`
	UpdatedAt        time.Time           `json:"updated_at"`
}

type GetBenchmarkListItem struct {
	Benchmark GetBenchmarkListMetadata     `json:"benchmark"`
	Findings  *GetBenchmarkDetailsFindings `json:"findings"`
}

type GetBenchmarkListResponse struct {
	Items      []GetBenchmarkListItem `json:"items"`
	TotalCount int                    `json:"total_count"`
}

type IntegrationInfo struct {
	Integration        string `json:"integration"`
	Type               string `json:"type"`
	ID                 string `json:"id"`
	IDName             string `json:"id_name"`
	IntegrationTracker string `json:"integration_tracker"`
}

type IntegrationFilter struct {
	Integration        *string `json:"integration"`
	ID                 *string `json:"id"`
	IDName             *string `json:"id_name"`
	IntegrationTracker *string `json:"integration_tracker"`
}

func GetTypeFromIntegration(integration string) string {
	switch strings.ToLower(integration) {
	case "aws":
		return "aws_account"
	case "azure":
		return "azure_subscription"
	default:
		return ""
	}
}

type IntegrationFilterRequest struct {
	Integration []IntegrationFilter `json:"integration"`
}

type SeveritySummary struct {
	Issues    int `json:"issues"`
	Resources int `json:"resources"`
	Controls  int `json:"controls"`
}
type TopIntegration struct {
	IntegrationInfo IntegrationInfo `json:"integration_info"`
	Issues          int             `json:"issues"`
}

type ComplianceSummaryOfIntegrationRequest struct {
	BenchmarkId string            `json:"benchmark_id"'`
	Integration IntegrationFilter `json:"integration"`
	ShowTop     int               `json:"show_top"`
}

type TopFiledRecordV2 struct {
	Field  string `json:"field"`
	Key    string `json:"key"`
	Issues int    `json:"issues"`
}

type ComplianceSummaryOfIntegrationResponse struct {
	BenchmarkID                string                             `json:"benchmark_id"`
	Integration                IntegrationInfo                    `json:"integration"`
	ComplianceScore            float64                            `json:"compliance_score"`
	SeveritySummaryByControl   BenchmarkControlsSeverityStatusV2  `json:"severity_summary_by_control"`
	SeveritySummaryByResource  BenchmarkResourcesSeverityStatusV2 `json:"severity_summary_by_resource"`
	FindingsSummary            ConformanceStatusSummaryV2         `json:"findings_summary"`
	IssuesCount                int                                `json:"issues_count"`
	TopResourcesWithIssues     []TopFiledRecordV2                 `json:"top_resources_with_issues"`
	TopResourceTypesWithIssues []TopFiledRecordV2                 `json:"top_resource_types_with_issues"`
	TopControlsWithIssues      []TopFiledRecordV2                 `json:"top_controls_with_issues"`
	LastEvaluatedAt            *time.Time                         `json:"last_evaluated_at"`
	LastJobStatus              string                             `json:"last_job_status"`
	LastJobId                  string                             `json:"last_job_id"`
}

type ComplianceSummaryOfBenchmarkRequest struct {
	Benchmarks []string `json:"benchmarks"`
	IsRoot     *bool    `json:"is_root"`
	ShowTop    int      `json:"show_top"`
}

type ComplianceSummaryOfBenchmarkResponse struct {
	BenchmarkID                string                             `json:"benchmark_id"`
	ComplianceScore            float64                            `json:"compliance_score"`
	SeveritySummaryByControl   BenchmarkControlsSeverityStatusV2  `json:"severity_summary_by_control"`
	SeveritySummaryByResource  BenchmarkResourcesSeverityStatusV2 `json:"severity_summary_by_resource"`
	FindingsSummary            ConformanceStatusSummaryV2         `json:"findings_summary"`
	IssuesCount                int                                `json:"issues_count"`
	TopIntegrations            []TopIntegration                   `json:"top_integrations"`
	TopResourcesWithIssues     []TopFiledRecordV2                 `json:"top_resources_with_issues"`
	TopResourceTypesWithIssues []TopFiledRecordV2                 `json:"top_resource_types_with_issues"`
	TopControlsWithIssues      []TopFiledRecordV2                 `json:"top_controls_with_issues"`
	LastEvaluatedAt            *time.Time                         `json:"last_evaluated_at"`
	LastJobStatus              string                             `json:"last_job_status"`
	LastJobId                  string                             `json:"last_job_id"`
}

type ListBenchmarksFiltersResponse struct {
	ParentBenchmarkID []string `json:"parent_benchmark_id"`
	PrimaryTable      []string `json:"primary_table"`
	ListOfTables      []string `json:"list_of_tables"`
}
