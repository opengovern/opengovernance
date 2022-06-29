package api

import (
	"github.com/google/uuid"
	compliance_report "gitlab.com/keibiengine/keibi-engine/pkg/compliance-report"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/kafka"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/api"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
)

type SourceType string

const (
	SourceCloudAWS   SourceType = "AWS"
	SourceCloudAzure SourceType = "Azure"
)

type DirectionType string

const (
	DirectionAscending  DirectionType = "asc"
	DirectionDescending DirectionType = "desc"
)

type SortFieldType string

const (
	SortFieldResourceID    SortFieldType = "resourceID"
	SortFieldName          SortFieldType = "name"
	SortFieldSourceType    SortFieldType = "provider"
	SortFieldResourceType  SortFieldType = "resourceType"
	SortFieldResourceGroup SortFieldType = "resourceGroup"
	SortFieldLocation      SortFieldType = "location"
	SortFieldSourceID      SortFieldType = "sourceID"
)

type GetResourceRequest struct {
	ResourceType string `json:"resourceType" validate:"required"`
	ID           string `json:"ID" validate:"required"` //	Resource ID
}

type LocationByProviderResponse struct {
	Name string `json:"name"`
}

type RunQueryRequest struct {
	Page api.PageRequest `json:"page" validate:"required"`
	// NOTE: we don't support multi-field sort for now, if sort is empty, results would be sorted by first column
	Sorts []SmartQuerySortItem `json:"sorts"`
}

type RunQueryResponse struct {
	Page    api.PageResponse `json:"page"`
	Headers []string         `json:"headers"` // column names
	// result of query. in order to access a specific cell please use Result[Row][Column]
	Result [][]interface{} `json:"result"`
}

type GetResourcesRequest struct {
	Query   string  `json:"query"`                       // search query
	Filters Filters `json:"filters" validate:"required"` // search filters
	// NOTE: we don't support multi-field sort for now, if sort is empty, results would be sorted by first column
	Sorts []ResourceSortItem `json:"sorts"`
	Page  api.PageRequest    `json:"page" validate:"required"`
}

// Filters model
// @Description if you provide two values for same filter OR operation would be used
// @Description if you provide value for two filters AND operation would be used
type Filters struct {
	// if you dont need to use this filter, leave them empty. (e.g. [])
	ResourceType []string `json:"resourceType"`
	// if you dont need to use this filter, leave them empty. (e.g. [])
	Location []string `json:"location"`
	// if you dont need to use this filter, leave them empty. (e.g. [])
	SourceID []string `json:"sourceID"`
}

type ResourceSortItem struct {
	Field     SortFieldType `json:"field" enums:"resourceID,name,provider,resourceType,resourceGroup,location,sourceID"`
	Direction DirectionType `json:"direction" enums:"asc,desc"`
}

type SmartQuerySortItem struct {
	// fill this with column name
	Field     string        `json:"field"`
	Direction DirectionType `json:"direction" enums:"asc,desc"`
}

type GetResourcesResponse struct {
	Resources []AllResource    `json:"resources"`
	Page      api.PageResponse `json:"page"`
}

type AllResource struct {
	Name             string     `json:"name"`
	Provider         SourceType `json:"provider"`
	ResourceType     string     `json:"resourceType"`
	ResourceTypeName string     `json:"resourceTypeName"`
	Location         string     `json:"location"`
	ResourceID       string     `json:"resourceID"`
	SourceID         string     `json:"sourceID"`
	SourceName       string     `json:"sourceName"`

	Attributes map[string]string `json:"attributes"`
}

type BenchmarkAssignment struct {
	BenchmarkId string `json:"benchmarkId"`
	SourceId    string `json:"sourceId"`
	AssignedAt  int64  `json:"assignedAt"`
}

type BenchmarkAssignedSource struct {
	SourceId   string `json:"sourceId"`
	AssignedAt int64  `json:"assignedAt"`
}

func (r AllResource) ToCSVRecord() []string {
	h := []string{r.Name, string(r.Provider), r.ResourceType, r.Location,
		r.ResourceID, r.SourceID}
	for _, value := range r.Attributes {
		h = append(h, value)
	}
	return h
}

func (r AllResource) ToCSVHeaders() []string {
	h := []string{"Name", "Provider", "ResourceType", "Location", "ResourceID", "SourceID"}
	for key := range r.Attributes {
		h = append(h, key)
	}
	return h
}

type GetAzureResourceResponse struct {
	Resources []AzureResource  `json:"resources"`
	Page      api.PageResponse `json:"page"`
}

type AzureResource struct {
	Name             string `json:"name"`
	ResourceType     string `json:"resourceType"`
	ResourceTypeName string `json:"resourceTypeName"`
	ResourceGroup    string `json:"resourceGroup"`
	Location         string `json:"location"`
	ResourceID       string `json:"resourceID"`
	SubscriptionID   string `json:"subscriptionID"`
	SubscriptionName string `json:"sourceName"`

	Attributes map[string]string `json:"attributes"`
}

func (r AzureResource) ToCSVRecord() []string {
	h := []string{r.Name, r.ResourceType, r.ResourceGroup, r.Location, r.ResourceID, r.SubscriptionID}
	for _, value := range r.Attributes {
		h = append(h, value)
	}
	return h
}
func (r AzureResource) ToCSVHeaders() []string {
	h := []string{"Name", "ResourceType", "ResourceGroup", "Location", "ResourceID", "SubscriptionID"}
	for key := range r.Attributes {
		h = append(h, key)
	}
	return h
}

type GetAWSResourceResponse struct {
	Resources []AWSResource    `json:"resources"`
	Page      api.PageResponse `json:"page"`
}

type AWSResource struct {
	Name             string `json:"name"`
	ResourceType     string `json:"resourceType"`
	ResourceTypeName string `json:"resourceTypeName"`
	ResourceID       string `json:"resourceID"`
	Region           string `json:"location"`
	AccountID        string `json:"accountID"`
	AccountName      string `json:"accountName"`

	Attributes map[string]string `json:"attributes"`
}

func (r AWSResource) ToCSVRecord() []string {
	h := []string{r.Name, r.ResourceType, r.ResourceID, r.Region, r.AccountID}
	for _, value := range r.Attributes {
		h = append(h, value)
	}
	return h
}
func (r AWSResource) ToCSVHeaders() []string {
	h := []string{"Name", "ResourceType", "ResourceID", "Region", "AccountID"}
	for key := range r.Attributes {
		h = append(h, key)
	}
	return h
}

type SummaryQueryResponse struct {
	Hits SummaryQueryHits `json:"hits"`
}
type SummaryQueryHits struct {
	Total keibi.SearchTotal `json:"total"`
	Hits  []SummaryQueryHit `json:"hits"`
}
type SummaryQueryHit struct {
	ID      string               `json:"_id"`
	Score   float64              `json:"_score"`
	Index   string               `json:"_index"`
	Type    string               `json:"_type"`
	Version int64                `json:"_version,omitempty"`
	Source  kafka.LookupResource `json:"_source"`
	Sort    []interface{}        `json:"sort"`
}

type GenericQueryResponse struct {
	Hits GenericQueryHits `json:"hits"`
}
type GenericQueryHits struct {
	Total keibi.SearchTotal `json:"total"`
	Hits  []GenericQueryHit `json:"hits"`
}
type GenericQueryHit struct {
	ID      string                 `json:"_id"`
	Score   float64                `json:"_score"`
	Index   string                 `json:"_index"`
	Type    string                 `json:"_type"`
	Version int64                  `json:"_version,omitempty"`
	Source  map[string]interface{} `json:"_source"`
	Sort    []interface{}          `json:"sort"`
}

type SmartQueryItem struct {
	ID          uint              `json:"id"`
	Provider    string            `json:"provider"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Query       string            `json:"query"`
	Tags        map[string]string `json:"tags"`
}

type TimeRangeFilter struct {
	From int64 // from epoch millisecond
	To   int64 // from epoch millisecond
}

type ComplianceReportFilters struct {
	TimeRange *TimeRangeFilter `json:"timeRange"`
	GroupID   *string          `json:"groupID"` // benchmark id or control id
}

type GetComplianceReportRequest struct {
	Filters    ComplianceReportFilters      `json:"filters"`
	ReportType compliance_report.ReportType `json:"reportType" enums:"benchmark,control,result"`
	Page       api.PageRequest              `json:"page" validate:"required"`
}

type GetComplianceReportResponse struct {
	Reports []compliance_report.Report `json:"reports"`
	Page    api.PageResponse           `json:"page"`
}

type BenchmarkState string

const (
	BenchmarkStateEnabled  = "enabled"
	BenchmarkStateDisabled = "disabled"
)

type Benchmark struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Provider    SourceType `json:"provider"`
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

type ListQueryRequest struct {
	TitleFilter    string      `json:"titleFilter"`
	ProviderFilter *SourceType `json:"providerFilter"`
	Labels         []string    `json:"labels"`
}

type TopAccountResponse struct {
	SourceID      string `json:"sourceID"`
	ResourceCount int    `json:"resourceCount"`
}

type TopAccountCostResponse struct {
	SourceID string  `json:"sourceID"`
	Cost     float64 `json:"cost"`
}

type TopServicesResponse struct {
	ServiceName      string `json:"serviceName"`
	ResourceCount    int    `json:"resourceCount"`
	LastDayCount     int    `json:"lastDayCount"`
	LastWeekCount    int    `json:"lastWeekCount"`
	LastQuarterCount int    `json:"lastQuarterCount"`
	LastYearCount    int    `json:"lastYearCount"`
}

type TopServiceCostResponse struct {
	ServiceName string  `json:"serviceName"`
	Cost        float64 `json:"cost"`
}

type CategoriesResponse struct {
	CategoryName     string `json:"serviceName"`
	ResourceCount    int    `json:"resourceCount"`
	LastDayCount     int    `json:"lastDayCount"`
	LastWeekCount    int    `json:"lastWeekCount"`
	LastQuarterCount int    `json:"lastQuarterCount"`
	LastYearCount    int    `json:"lastYearCount"`
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

type ServiceDistributionItem struct {
	ServiceName  string         `json:"serviceName"`
	Distribution map[string]int `json:"distribution"`
}
