package api

import (
	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
	"gitlab.com/keibiengine/keibi-engine/pkg/types"
)

type BenchmarkAssignment struct {
	BenchmarkId string `json:"benchmarkId"`
	SourceId    string `json:"sourceId"`
	AssignedAt  int64  `json:"assignedAt"`
}

type BenchmarkAssignedSource struct {
	Connection types.FullConnection `json:"connection"`
	AssignedAt int64                `json:"assignedAt"`
}

type BenchmarkState string

const (
	BenchmarkStateEnabled  = "enabled"
	BenchmarkStateDisabled = "disabled"
)

type Benchmark struct {
	ID          string      `json:"id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Provider    source.Type `json:"provider"`
	State       BenchmarkState
	Tags        map[string]string `json:"tags"`
}

type GetBenchmarkTag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Count int    `json:"count"`
}

type GetBenchmarkDetailsResponse struct {
	Categories    []string `json:"categories"`
	Subcategories []string `json:"subcategories"`
	Sections      []string `json:"sections"`
}

type Policy struct {
	ID                    string            `json:"id"`
	Title                 string            `json:"title"`
	Description           string            `json:"description"`
	Category              string            `json:"category"`
	Subcategory           string            `json:"subcategory"`
	Section               string            `json:"section"`
	Severity              string            `json:"severity"`
	Provider              string            `json:"provider"`
	ManualVerification    string            `json:"manualVerification"`
	ManualRemedation      string            `json:"manualRemedation"`
	CommandLineRemedation string            `json:"commandLineRemedation"`
	QueryToRun            string            `json:"queryToRun"`
	Tags                  map[string]string `json:"tags"`
}

type PolicyResultStatus string

const (
	PolicyResultStatusFailed PolicyResultStatus = "failed"
	PolicyResultStatusPassed PolicyResultStatus = "passed"
)

type PolicyResult struct {
	ID                 string             `json:"id"`
	Title              string             `json:"title"`
	Category           string             `json:"category"`
	Subcategory        string             `json:"subcategory"`
	Section            string             `json:"section"`
	Severity           string             `json:"severity"`
	Provider           string             `json:"provider"`
	Status             PolicyResultStatus `json:"status" enums:"passed,failed"`
	CompliantResources int                `json:"compliantResources"`
	TotalResources     int                `json:"totalResources"`
	DescribedAt        int64              `json:"describedAt"`
	CreatedAt          int64              `json:"createdAt"`
}

type ResultPolicy struct {
	ID          string             `json:"id"`
	Title       string             `json:"title"`
	Category    string             `json:"category"`
	Subcategory string             `json:"subcategory"`
	Section     string             `json:"section"`
	Severity    string             `json:"severity"`
	Provider    string             `json:"provider"`
	Status      PolicyResultStatus `json:"status" enums:"passed,failed"`
	DescribedAt int64              `json:"describedAt"`
	CreatedAt   int64              `json:"createdAt"`
}

type ResultCompliancy struct {
	ID                 string             `json:"id"`
	Title              string             `json:"title"`
	Category           string             `json:"category"`
	Subcategory        string             `json:"subcategory"`
	Section            string             `json:"section"`
	Severity           string             `json:"severity"`
	Provider           string             `json:"provider"`
	Status             PolicyResultStatus `json:"status" enums:"passed,failed"`
	ResourcesWithIssue int                `json:"resourcesWithIssue"`
	TotalResources     int                `json:"totalResources"`
}

type ResultPolicyResourceSummary struct {
	ResourcesByLocation       map[string]int `json:"resourcesByLocation"`
	CompliantResourceCount    int            `json:"compliantResourceCount"`
	NonCompliantResourceCount int            `json:"nonCompliantResourceCount"`
}

type ComplianceTrendDataPoint struct {
	Timestamp      int64 `json:"timestamp"`
	Compliant      int64 `json:"compliant"`
	TotalResources int64 `json:"totalResources"`
}

type TrendDataPoint struct {
	Timestamp int64 `json:"timestamp"`
	Value     int64 `json:"value"`
}

type BenchmarkAccountComplianceResponse struct {
	TotalCompliantAccounts    int `json:"totalCompliantAccounts"`
	TotalNonCompliantAccounts int `json:"totalNonCompliantAccounts"`
}

type BenchmarkScoreResponse struct {
	BenchmarkID       string `json:"benchmarkID"`
	NonCompliantCount int    `json:"nonCompliantCount"`
}

type AccountCompliancyResponse struct {
	SourceID       uuid.UUID `json:"sourceID"`
	TotalResources int       `json:"totalResources"`
	TotalCompliant int       `json:"totalCompliant"`
}

type ServiceCompliancyResponse struct {
	ServiceName    string `json:"serviceName"`
	TotalResources int    `json:"totalResources"`
	TotalCompliant int    `json:"totalCompliant"`
}

type FindingFilters struct {
	Provider       []source.Type `json:"provider"`
	ResourceTypeID []string      `json:"resourceTypeID"`
	ConnectionID   []uuid.UUID   `json:"connectionID"`
	FindingStatus  []es.Status   `json:"findingStatus"`
	BenchmarkID    []string      `json:"benchmarkID"`
	PolicyID       []string      `json:"policyID"`
	Severity       []string      `json:"severity"`
}

type FindingResponseFilters struct {
	Provider      []source.Type            `json:"provider"`
	ResourceType  []types.FullResourceType `json:"resourceTypeID"`
	Connections   []types.FullConnection   `json:"connections"`
	FindingStatus []types.ComplianceResult `json:"findingStatus"`
	Benchmarks    []types.FullBenchmark    `json:"benchmarks"`
	Policies      []types.FullPolicy       `json:"policies"`
	Severity      []string                 `json:"severity"`
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
	Field     SortFieldType `json:"field" enums:"resourceID,resourceName,resourceType,serviceName,category,resourceLocation,status,describedAt,evaluatedAt,sourceID,connectionProviderID,connectionProviderName,sourceType,benchmarkID,policyID,policySeverity"`
	Direction DirectionType `json:"direction" enums:"asc,desc"`
}

type Page struct {
	No   int `json:"no,omitempty"`
	Size int `json:"size,omitempty"`
}

type GetFindingsRequest struct {
	Filters FindingFilters    `json:"filters"`
	Sorts   []FindingSortItem `json:"sorts"`
	Page    Page              `json:"page" validate:"required"`
}

type GetFindingsResponse struct {
	Findings   []es.Finding `json:"findings"`
	TotalCount int64        `json:"totalCount,omitempty"`
}

type GetFindingsFiltersResponse struct {
	Filters FindingResponseFilters `json:"filters"`
}

type Datapoint struct {
	Time  int64 `json:"time"`
	Value int64 `json:"value"`
}

type StatusCount struct {
	Passed int64 `json:"passed"`
	Failed int64 `json:"failed"`
}

type BenchmarkSummaryPolicySummary struct {
	Policy       types.FullPolicy                   `json:"policy"`
	ShortSummary types.ComplianceResultShortSummary `json:"shortSummary"`
}

type BenchmarkSummaryResourceSummary struct {
	Resource     types.FullResource                 `json:"resource"`
	ShortSummary types.ComplianceResultShortSummary `json:"shortSummary"`
}

type BenchmarkSummary struct {
	Title                    string                             `json:"title"`
	Description              string                             `json:"description"`
	ShortSummary             types.ComplianceResultShortSummary `json:"shortSummary"`
	Policies                 []BenchmarkSummaryPolicySummary    `json:"policies"`
	Resources                []BenchmarkSummaryResourceSummary  `json:"resources"`
	CompliancyTrend          []Datapoint                        `json:"trend"`
	AssignedConnectionsCount int64                              `json:"assignedConnectionsCount"`
	TotalConnectionResources int64                              `json:"totalConnectionResources"`
	Tags                     map[string]string                  `json:"tags"`
	Enabled                  bool                               `json:"enabled"`
}

type BenchmarkSummaryConnectionSummary struct {
	Connection   types.FullConnection               `json:"connection"`
	ShortSummary types.ComplianceResultShortSummary `json:"shortSummary"`
}

type GetBenchmarksSummaryResponse struct {
	ShortSummary types.ComplianceResultShortSummary  `json:"shortSummary"`
	TotalAssets  int                                 `json:"totalAssets"`
	Connections  []BenchmarkSummaryConnectionSummary `json:"connections"`
	Benchmarks   []BenchmarkSummary                  `json:"benchmarks"`
}

type PolicySummary struct {
	Title       string             `json:"title"`
	Category    string             `json:"category"`
	Subcategory string             `json:"subcategory"`
	Severity    types.Severity     `json:"severity"`
	Status      types.PolicyStatus `json:"status"`
	CreatedAt   int64              `json:"createdAt"`
}

type GetPoliciesSummaryResponse struct {
	BenchmarkTitle       string                        `json:"title"`
	BenchmarkDescription string                        `json:"description"`
	ComplianceSummary    types.ComplianceResultSummary `json:"complianceSummary"`
	PolicySummary        []PolicySummary               `json:"policySummary"`
	Tags                 map[string]string             `json:"tags"`
	Enabled              bool                          `json:"enabled"`
}

type GetFindingsMetricsResponse struct {
	TotalFindings   types.HistoricalCount `json:"totalFindings"`
	FailedFindings  types.HistoricalCount `json:"failedFindings"`
	PassedFindings  types.HistoricalCount `json:"passedFindings"`
	UnknownFindings types.HistoricalCount `json:"unknownFindings"`
}

type GetFindingDetailsResponse struct {
	Connection        types.FullConnection   `json:"connection"`
	Resource          types.FullResource     `json:"resource"`
	ResourceType      types.FullResourceType `json:"resourceType"`
	State             types.ComplianceResult `json:"state"`
	CreatedAt         int64                  `json:"createdAt"`
	PolicyTags        map[string]string      `json:"policyTags"`
	PolicyDescription string                 `json:"policyDescription"`
	Reason            string                 `json:"reason"`
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
