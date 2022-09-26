package api

import (
	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
)

type BenchmarkAssignment struct {
	BenchmarkId string `json:"benchmarkId"`
	SourceId    string `json:"sourceId"`
	AssignedAt  int64  `json:"assignedAt"`
}

type BenchmarkAssignedSource struct {
	SourceId   string `json:"sourceId"`
	AssignedAt int64  `json:"assignedAt"`
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
	SourceID       []uuid.UUID   `json:"sourceID"`
	FindingStatus  []es.Status   `json:"findingStatus"`
	BenchmarkID    []string      `json:"benchmarkID"`
	Severity       []string      `json:"severity"`
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
	Filters FindingFilters `json:"filters"`
}
