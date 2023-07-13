package api

import (
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/es"
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type BenchmarkAssignment struct {
	BenchmarkId  string `json:"benchmarkId" example:"azure_cis_v140"`                    // Benchmark ID
	ConnectionId string `json:"sourceId" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"` // Connection ID
	AssignedAt   int64  `json:"assignedAt"`                                              // Unix timestamp
}

type BenchmarkAssignedSource struct {
	ConnectionID   string      `json:"connectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"` // Connection ID
	ConnectionName string      `json:"connectionName"`                                              // Connection Name
	Connector      source.Type `json:"connector" example:"Azure"`                                   // Clout Provider
	Status         bool        `json:"status" example:"true"`                                       // Status
}

type FindingFilters struct {
	Connector      []source.Type            `json:"connector" example:"Azure"`                                                                                    // Clout Provider
	ResourceID     []string                 `json:"resourceID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1"` // Resource unique identifier
	ResourceTypeID []string                 `json:"resourceTypeID" example:"/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines"`  // Resource type
	ConnectionID   []string                 `json:"connectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`                                                  // Connection ID
	BenchmarkID    []string                 `json:"benchmarkID" example:"azure_cis_v140"`                                                                         // Benchmark ID
	PolicyID       []string                 `json:"policyID" example:"azure_cis_v140_7_5"`                                                                        // Policy ID
	Severity       []string                 `json:"severity" example:"low"`                                                                                       // Severity
	Status         []types.ComplianceResult `json:"status" example:"alarm"`                                                                                       // Compliance result status
}

type FindingResponseFilters struct {
	Provider      []source.Type            `json:"provider" example:"Azure"`      // Clout Provider
	ResourceType  []types.FullResourceType `json:"resourceTypeID"`                // Resource type
	Connections   []types.FullConnection   `json:"connections"`                   // Connections
	FindingStatus []types.ComplianceResult `json:"findingStatus" example:"alarm"` // Compliance result status
	Benchmarks    []types.FullBenchmark    `json:"benchmarks"`                    // Benchmarks
	Policies      []types.FullPolicy       `json:"policies"`                      // Policies
	Severity      []string                 `json:"severity" example:"low"`        // Severity
}

type DirectionType string

const (
	DirectionAscending  DirectionType = "asc"
	DirectionDescending DirectionType = "desc"
)

type SortFieldType string

const (
	FieldResourceID             SortFieldType = "resourceID"
	FieldResourceName           SortFieldType = "resourceName"
	FieldResourceType           SortFieldType = "resourceType"
	FieldServiceName            SortFieldType = "serviceName"
	FieldCategory               SortFieldType = "category"
	FieldResourceLocation       SortFieldType = "resourceLocation"
	FieldStatus                 SortFieldType = "status"
	FieldDescribedAt            SortFieldType = "describedAt"
	FieldEvaluatedAt            SortFieldType = "evaluatedAt"
	FieldSourceID               SortFieldType = "sourceID"
	FieldConnectionProviderID   SortFieldType = "connectionProviderID"
	FieldConnectionProviderName SortFieldType = "connectionProviderName"
	FieldSourceType             SortFieldType = "sourceType"
	FieldBenchmarkID            SortFieldType = "benchmarkID"
	FieldPolicyID               SortFieldType = "policyID"
	FieldPolicySeverity         SortFieldType = "policySeverity"
)

type FindingSortItem struct {
	Field     SortFieldType `json:"field" enums:"resourceID,resourceName,resourceType,serviceName,category,resourceLocation,status,describedAt,evaluatedAt,sourceID,connectionProviderID,connectionProviderName,sourceType,benchmarkID,policyID,policySeverity" example:"status"` // Field to sort by
	Direction DirectionType `json:"direction" enums:"asc,desc" example:"asc"`                                                                                                                                                                                                     // Sort direction
}

type Page struct {
	No   int `json:"no,omitempty" example:"5"`     // Number of pages
	Size int `json:"size,omitempty" example:"100"` // Number of items per page
}

type GetFindingsRequest struct {
	Filters FindingFilters    `json:"filters"`
	Sorts   []FindingSortItem `json:"sorts"`
	Page    Page              `json:"page" validate:"required"`
}

type TopField = string

const (
	TopField_ResourceType TopField = "resourceType"
	TopField_CloudService TopField = "serviceName"
	TopField_CloudAccount TopField = "sourceID"
	TopField_Resources    TopField = "resourceID"
)

type GetTopFieldRequest struct {
	Field   TopField       `json:"field" enums:"resourceType,serviceName,sourceID,resourceID" example:"resourceType"` // Field to get top values for
	Filters FindingFilters `json:"filters"`
	Count   int            `json:"count" example:"1"` // Number of items to return
}

type TopFieldRecord struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

type GetTopFieldResponse struct {
	Records []TopFieldRecord `json:"records"`
}

type GetFindingsResponse struct {
	Findings   []es.Finding `json:"findings"`
	TotalCount int64        `json:"totalCount" example:"100"`
}

type GetFindingsFiltersResponse struct {
	Filters FindingResponseFilters `json:"filters"` // Filters
}

type Datapoint struct {
	Time  int64 `json:"time"`  // Time
	Value int64 `json:"value"` // Value
}

type GetBenchmarksSummaryResponse struct {
	BenchmarkSummary []BenchmarkEvaluationSummary `json:"benchmarkSummary"`

	TotalResult types.ComplianceResultSummary `json:"totalResult"`
	TotalChecks types.SeverityResult          `json:"totalChecks"`
}

type BenchmarkEvaluationSummary struct {
	ID          string                        `json:"id" example:"azure_cis_v140"`                                                                                                                                                       // Benchmark ID
	Title       string                        `json:"title" example:"Azure CIS v1.4.0"`                                                                                                                                                  // Benchmark title
	Description string                        `json:"description" example:"The CIS Microsoft Azure Foundations Security Benchmark provides prescriptive guidance for establishing a secure baseline configuration for Microsoft Azure."` // Benchmark description
	Connectors  []source.Type                 `json:"connectors" example:"[Azure]"`                                                                                                                                                      // Cloud providers
	Tags        map[string][]string           `json:"tags" `                                                                                                                                                                             // Tags
	Enabled     bool                          `json:"enabled" example:"true"`                                                                                                                                                            // Enabled
	Result      types.ComplianceResultSummary `json:"result"`                                                                                                                                                                            // Compliance result summary
	Checks      types.SeverityResult          `json:"checks"`                                                                                                                                                                            // Checks summary
}

type ResultDatapoint struct {
	Time   int64                `json:"time"`   // Datapoint Time
	Result types.SeverityResult `json:"result"` // Result
}

type BenchmarkResultTrend struct {
	ResultDatapoint []ResultDatapoint `json:"resultTrend"`
}

type PolicyTree struct {
	ID          string             `json:"id" example:"azure_cis_v140_7_5"`                                                            // Policy ID
	Title       string             `json:"title" example:"7.5 Ensure that the latest OS Patches for all Virtual Machines are applied"` // Policy title
	Severity    types.Severity     `json:"severity" example:"low"`                                                                     // Severity
	Status      types.PolicyStatus `json:"status" example:"passed"`                                                                    // Status
	LastChecked int64              `json:"lastChecked" example:"0"`                                                                    // Last checked
}

type BenchmarkTree struct {
	ID    string `json:"id" example:"azure_cis_v140"` // Benchmark ID
	Title string `json:"title" example:"CIS v1.4.0"`  // Benchmark title

	Children []BenchmarkTree `json:"children"`
	Policies []PolicyTree    `json:"policies"`
}

type GetFindingsMetricsResponse struct {
	TotalFindings   int64 `json:"totalFindings" example:"100"`
	FailedFindings  int64 `json:"failedFindings" example:"10"`
	PassedFindings  int64 `json:"passedFindings" example:"90"`
	UnknownFindings int64 `json:"unknownFindings" example:"0"`

	LastTotalFindings   int64 `json:"lastTotalFindings" example:"100"`
	LastFailedFindings  int64 `json:"lastFailedFindings" example:"10"`
	LastPassedFindings  int64 `json:"lastPassedFindings" example:"90"`
	LastUnknownFindings int64 `json:"lastUnknownFindings" example:"0"`
}

type Alarms struct {
	Policy    types.FullPolicy       `json:"policy"`
	CreatedAt int64                  `json:"created_at" example:"2023-04-21T08:53:09.928Z"`
	Status    types.ComplianceResult `json:"status" example:"alarm"`
}

type GetFindingDetailsResponse struct {
	Connection        types.FullConnection   `json:"connection"`
	Resource          types.FullResource     `json:"resource"`
	ResourceType      types.FullResourceType `json:"resourceType"`
	State             types.ComplianceResult `json:"state" example:"alarm"`
	CreatedAt         int64                  `json:"createdAt" example:"2023-04-21T08:53:09.928Z"`
	PolicyTags        map[string]string      `json:"policyTags"`
	PolicyDescription string                 `json:"policyDescription"`
	Reason            string                 `json:"reason"`
	Alarms            []Alarms               `json:"alarms"`
}

type InsightRecord struct {
	Name  string `json:"name"`
	Value int64  `json:"value"`
}

type GetBenchmarkInsightResponse struct {
	TopResourceType []InsightRecord `json:"topResourceType"`
	TopCategory     []InsightRecord `json:"topCategory"`
	TopAccount      []InsightRecord `json:"topAccount"`
	Severity        []InsightRecord `json:"severity"`
}
