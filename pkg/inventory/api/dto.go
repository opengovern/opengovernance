package api

import (
	"time"

	"github.com/kaytu-io/kaytu-util/pkg/source"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/es"

	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
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
	SortFieldName          SortFieldType = "resourceName"
	SortFieldSourceType    SortFieldType = "provider"
	SortFieldResourceType  SortFieldType = "resourceType"
	SortFieldResourceGroup SortFieldType = "resourceGroup"
	SortFieldLocation      SortFieldType = "location"
	SortFieldSourceID      SortFieldType = "connectionID"
)

type CostWithUnit struct {
	Cost float64 `json:"cost"` // Value
	Unit string  `json:"unit"` // Currency
}

type GetResourceRequest struct {
	ResourceType string `json:"resourceType" validate:"required"` // Resource ID
	ID           string `json:"ID" validate:"required"`           // Resource ID
}

type LocationByProviderResponse struct {
	Name string `json:"name"` // Name of the region
}

type RunQueryRequest struct {
	Page Page `json:"page" validate:"required"`
	// NOTE: we don't support multi-field sort for now, if sort is empty, results would be sorted by first column
	Sorts []SmartQuerySortItem `json:"sorts"`
}

type RunQueryResponse struct {
	Title   string          `json:"title"`   // Query Title
	Query   string          `json:"query"`   // Query
	Headers []string        `json:"headers"` // Column names
	Result  [][]interface{} `json:"result"`  // Result of query. in order to access a specific cell please use Result[Row][Column]
}

type Page struct {
	No   int `json:"no,omitempty"`
	Size int `json:"size,omitempty"`
}

type GetResourcesRequest struct {
	Query   string  `json:"query"`                       // search query
	Filters Filters `json:"filters" validate:"required"` // search filters
	// NOTE: we don't support multi-field sort for now, if sort is empty, results would be sorted by first column
	Sorts []ResourceSortItem `json:"sorts"`
	Page  Page               `json:"page" validate:"required"`
}

// Filters model
//
//	@Description	if you provide two values for same filter OR operation would be used
//	@Description	if you provide value for two filters AND operation would be used
type Filters struct {
	// if you dont need to use this filter, leave them empty. (e.g. [])
	ResourceType []string `json:"resourceType"`
	// if you dont need to use this filter, leave them empty. (e.g. [])
	Category []string `json:"category"`
	// if you dont need to use this filter, leave them empty. (e.g. [])
	Service []string `json:"service"`
	// if you dont need to use this filter, leave them empty. (e.g. [])
	Location []string `json:"location"`
	// if you dont need to use this filter, leave them empty. (e.g. [])
	SourceID []string `json:"sourceID"`
	// if you dont need to use this filter, leave them empty. (e.g. {})
	Tags map[string]string `json:"tags"`
}

// ResourceFilters model
//
//	@Description	if you provide two values for same filter OR operation would be used
//	@Description	if you provide value for two filters AND operation would be used
type ResourceFilters struct {
	// if you dont need to use this filter, leave them empty. (e.g. [])
	ResourceType []string `json:"resourceType"`
	// if you dont need to use this filter, leave them empty. (e.g. [])
	Category []string `json:"category"`
	// if you dont need to use this filter, leave them empty. (e.g. [])
	Service []string `json:"service"`
	// if you dont need to use this filter, leave them empty. (e.g. [])
	Location []string `json:"location"`
	// if you dont need to use this filter, leave them empty. (e.g. [])
	Provider []string `json:"provider"`
	// if you dont need to use this filter, leave them empty. (e.g. [])
	Connections []string `json:"connections"`
	// if you dont need to use this filter, leave them empty. (e.g. [])
	TagKeys []string `json:"tagKeys"`
	// if you dont need to use this filter, leave them empty. (e.g. [])
	TagValues map[string][]string `json:"tagValues"`
}

type ResourceTypeFull struct {
	ResourceTypeARN  string `json:"resource_type_arn"`
	ResourceTypeName string `json:"resource_type_name"`
}
type ConnectionFull struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type ResourceFiltersResponse struct {
	// if you dont need to use this filter, leave them empty. (e.g. [])
	ResourceType []ResourceTypeFull `json:"resourceType"`
	// if you dont need to use this filter, leave them empty. (e.g. [])
	Category map[string]string `json:"category"`
	// if you dont need to use this filter, leave them empty. (e.g. [])
	Service map[string]string `json:"service"`
	// if you dont need to use this filter, leave them empty. (e.g. [])
	Location []string `json:"location"`
	// if you dont need to use this filter, leave them empty. (e.g. [])
	Provider []string `json:"provider"`
	// if you dont need to use this filter, leave them empty. (e.g. [])
	Connections []ConnectionFull `json:"connections"`
	// if you dont need to use this filter, leave them empty. (e.g. [])
	TagKeys []string `json:"tagKeys"`
	// if you dont need to use this filter, leave them empty. (e.g. [])
	TagValues map[string][]string `json:"tagValues"`
}

type GetFiltersRequest struct {
	Query   string          `json:"query"`                       // search query
	Filters ResourceFilters `json:"filters" validate:"required"` // search filters
}

type GetFiltersResponse struct {
	Filters ResourceFiltersResponse `json:"filters"` // search filters
}

type ResourceSortItem struct {
	Field     SortFieldType `json:"field" enums:"resourceID,resourceName,provider,resourceType,resourceGroup,location,connectionID"`
	Direction DirectionType `json:"direction" enums:"asc,desc"`
}

type SmartQuerySortItem struct {
	// fill this with column name
	Field     string        `json:"field"`
	Direction DirectionType `json:"direction" enums:"asc,desc"`
}

type GetResourcesResponse struct {
	Resources  []AllResource `json:"resources"`            // A list of AWS resources with details
	TotalCount int64         `json:"totalCount,omitempty"` // Number of returned resources
}

type AllResource struct {
	ResourceName           string     `json:"resourceName"`           // Resource Name
	ResourceID             string     `json:"resourceID"`             // Resource Id
	ResourceType           string     `json:"resourceType"`           // Resource Type
	ResourceTypeName       string     `json:"resourceTypeName"`       // Resource Type Name
	ResourceCategory       string     `json:"resourceCategory"`       // Resource Category
	Provider               SourceType `json:"provider"`               // Resource Provider
	Location               string     `json:"location"`               // The Region of the resource
	ProviderConnectionID   string     `json:"providerConnectionID"`   // Provider Connection Id
	ProviderConnectionName string     `json:"providerConnectionName"` // Provider Connection Name

	Attributes map[string]string `json:"attributes"`
}

func (r AllResource) ToCSVRecord() []string {
	h := []string{r.ResourceName, string(r.Provider), r.ResourceTypeName, r.Location,
		r.ResourceID, r.ProviderConnectionID}
	for _, value := range r.Attributes {
		h = append(h, value)
	}
	return h
}

func (r AllResource) ToCSVHeaders() []string {
	h := []string{"Name", "Provider", "ResourceType", "Location", "ResourceID", "ProviderAccountID"}
	for key := range r.Attributes {
		h = append(h, key)
	}
	return h
}

type GetAzureResourceResponse struct {
	Resources  []AzureResource `json:"resources"`            // A list of Azure resources with details
	TotalCount int64           `json:"totalCount,omitempty"` // Number of returned resources
}

type AzureResource struct {
	ResourceName           string `json:"resourceName"`           // Resource Name
	ResourceID             string `json:"resourceID"`             // Resource Id
	ResourceType           string `json:"resourceType"`           // Resource Type
	ResourceTypeName       string `json:"resourceTypeName"`       // Resource Type Name
	ResourceCategory       string `json:"resourceCategory"`       // Resource Category
	ResourceGroup          string `json:"resourceGroup"`          // Resource Group
	Location               string `json:"location"`               // The Region of the resource
	ProviderConnectionID   string `json:"providerConnectionID"`   // Provider Connection Id
	ProviderConnectionName string `json:"providerConnectionName"` // Provider Connection Name

	Attributes map[string]string `json:"attributes"`
}

func (r AzureResource) ToCSVRecord() []string {
	h := []string{r.ResourceName, r.ResourceTypeName, r.ResourceGroup, r.Location, r.ResourceID, r.ProviderConnectionID}
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
	Resources  []AWSResource `json:"resources"`            // A list of AWS resources with details
	TotalCount int64         `json:"totalCount,omitempty"` // Number of returned resources
}

type AWSResource struct {
	ResourceName           string `json:"resourceName"`
	ResourceID             string `json:"resourceID"`
	ResourceType           string `json:"resourceType"`
	ResourceTypeName       string `json:"resourceTypeName"`
	ResourceCategory       string `json:"resourceCategory"`
	Location               string `json:"location"`
	ProviderConnectionID   string `json:"providerConnectionID"`
	ProviderConnectionName string `json:"providerConnectionName"`

	Attributes map[string]string `json:"attributes"`
}

func (r AWSResource) ToCSVRecord() []string {
	h := []string{r.ResourceName, r.ResourceTypeName, r.ResourceID, r.Location, r.ProviderConnectionID}
	for _, value := range r.Attributes {
		h = append(h, value)
	}
	return h
}
func (r AWSResource) ToCSVHeaders() []string {
	h := []string{"ResourceName", "ResourceType", "ResourceID", "Location", "ProviderConnectionID"}
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
	ID      string            `json:"_id"`
	Score   float64           `json:"_score"`
	Index   string            `json:"_index"`
	Type    string            `json:"_type"`
	Version int64             `json:"_version,omitempty"`
	Source  es.LookupResource `json:"_source"`
	Sort    []interface{}     `json:"sort"`
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
	ID          uint              `json:"id"`          // Query Id
	Provider    string            `json:"provider"`    // Provider
	Title       string            `json:"title"`       // Title
	Description string            `json:"description"` // Description
	Category    string            `json:"category"`    // Category (Tags[category])
	Query       string            `json:"query"`       // Query
	Tags        map[string]string `json:"tags"`        // Tags
}

type ListQueryRequest struct {
	TitleFilter    string      `json:"titleFilter"`    // Specifies the Title
	ProviderFilter *SourceType `json:"providerFilter"` // Specifies the Provider
	Labels         []string    `json:"labels"`         // Labels
}

type ConnectionResourceCountResponse struct {
	SourceID                string      `json:"sourceID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`              // Source Id
	Connector               source.Type `json:"connector" example:"azure"`                                            // Source Type
	ConnectorConnectionName string      `json:"connectorConnectionName" example:"example-account"`                    // Provider Connection Name
	ConnectorConnectionID   string      `json:"connectorConnectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"` // Provider Connection Id
	LifecycleState          string      `json:"lifecycleState" example:"enabled"`                                     // Lifesycle State
	ResourceCount           int         `json:"resourceCount" example:"100"`                                          // Number of resources
	OnboardDate             time.Time   `json:"onboardDate" example:"2023-05-22T12:50:22.499961Z"`                    // Onboard Date
	LastInventory           time.Time   `json:"lastInventory" example:"2023-05-22T12:50:22.499961Z"`                  // Last Inventory Date
}

type ConnectionData struct {
	ConnectionID  string     `json:"connectionID"`
	Count         int        `json:"count"`
	LastInventory *time.Time `json:"lastInventory"`
	Cost          float64    `json:"cost"`
}

type TopAccountResponse struct {
	SourceID               string `json:"sourceID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`             // Source Id
	Provider               string `json:"provider" example:"azure"`                                            // Account Provider
	ProviderConnectionName string `json:"providerConnectionName" example:"example-account"`                    // Account Provider Connection Name
	ProviderConnectionID   string `json:"providerConnectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"` // Account Provider Connection ID
	ResourceCount          int    `json:"resourceCount" example:"100"`                                         // Last number of Resources of the account
}

type TopAccountCostResponse struct {
	SourceID               string  `json:"sourceID"`               // Source Id
	ProviderConnectionName string  `json:"providerConnectionName"` // Account Provider Connection Name
	ProviderConnectionID   string  `json:"providerConnectionID"`   // Account Provider Connection ID
	Cost                   float64 `json:"cost"`                   // Account costs
}

type TopServicesResponse struct {
	ServiceName      string `json:"serviceName"`      // Service Name
	Provider         string `json:"provider"`         // Service Provider Name
	ResourceCount    int    `json:"resourceCount"`    // Number of resources
	LastDayCount     *int   `json:"lastDayCount"`     // Number of resources on last day
	LastWeekCount    *int   `json:"lastWeekCount"`    // Number of resources on last week
	LastQuarterCount *int   `json:"lastQuarterCount"` // Number of resources on last quarter
	LastYearCount    *int   `json:"lastYearCount"`    // Number of resources on last year
}

type TopServiceCostResponse struct {
	ServiceName string  `json:"serviceName"` // Service Name
	Cost        float64 `json:"cost"`        // Service Cost
}

type ResourceTypeResponse struct {
	ResourceType     string `json:"resourceType"`     // Resource Type
	ResourceTypeName string `json:"resourceTypeName"` // Resoutce Type Name
	ResourceCount    int    `json:"resourceCount"`    // Number of resources
	LastDayCount     *int   `json:"lastDayCount"`     // Number of resources on last day
	LastWeekCount    *int   `json:"lastWeekCount"`    // Number of resources on last week
	LastQuarterCount *int   `json:"lastQuarterCount"` // Number of resources on last quarter
	LastYearCount    *int   `json:"lastYearCount"`    // Number of resources on last year
}

type CategorizedMetricsResponse struct {
	Category map[string][]ResourceTypeResponse `json:"category"`
}

type LocationResponse struct {
	Location                   string   `json:"location"`                             // Region
	ResourceCount              *int     `json:"resourceCount,omitempty"`              // Number of resources in the region
	ResourceCountChangePercent *float64 `json:"resourceCountChangePercent,omitempty"` // Change in number of resources in the region
}

type RegionsByResourceCountResponse struct {
	TotalCount int                `json:"totalCount"`
	Regions    []LocationResponse `json:"regions"`
}

type MetricsResponse struct {
	MetricsName      string `json:"metricsName"`
	Value            int    `json:"value"`
	LastDayValue     *int   `json:"lastDayValue"`
	LastWeekValue    *int   `json:"lastWeekValue"`
	LastQuarterValue *int   `json:"lastQuarterValue"`
	LastYearValue    *int   `json:"lastYearValue"`
}

type ServiceDistributionItem struct {
	ServiceName  string         `json:"serviceName"`  // Service name
	Distribution map[string]int `json:"distribution"` // Distribution name
}

type Category struct {
	Name        string   `json:"name"`        // Category Name
	SubCategory []string `json:"subCategory"` // List of sub categories
}

type ConnectionSummaryCategory struct {
	ResourceCount int            `json:"resourceCount"`
	SubCategories map[string]int `json:"subCategories"`
}

type CategoriesMetrics struct {
	Categories map[string]CategoryMetric `json:"categories"`
}

type CategoryMetric struct {
	ResourceCount    int  `json:"resourceCount"`
	LastDayValue     *int `json:"lastDayValue"`
	LastWeekValue    *int `json:"lastWeekValue"`
	LastQuarterValue *int `json:"lastQuarterValue"`
	LastYearValue    *int `json:"lastYearValue"`

	SubCategories map[string]HistoryCount `json:"subCategories"`
}

type HistoryCount struct {
	Count            int  `json:"count"`
	LastDayValue     *int `json:"lastDayValue"`
	LastWeekValue    *int `json:"lastWeekValue"`
	LastQuarterValue *int `json:"lastQuarterValue"`
	LastYearValue    *int `json:"lastYearValue"`
}

type ConnectionSummaryResponse struct {
	Categories    map[string]ConnectionSummaryCategory `json:"categories"`    // Categories with their Summary
	CloudServices map[string]int                       `json:"cloudServices"` // Services as Key, Number of them as Value
	ResourceTypes map[string]int                       `json:"resourceTypes"` // Resource types as Key, Number of them as Value
}

type ListServiceSummariesResponse struct {
	TotalCount int              `json:"totalCount" example:"20"` // Number of services
	Services   []ServiceSummary `json:"services"`                // A list of service summeries
}

type ServiceSummary struct {
	Connector     source.Type `json:"connector" example:"Azure"`             // Cloud provider
	ServiceLabel  string      `json:"serviceLabel" example:"Compute"`        // Service Label
	ServiceName   string      `json:"serviceName" example:"compute"`         // Service Name
	ResourceCount *int        `json:"resourceCount,omitempty" example:"100"` // Number of Resources
}

type ListResourceTypesResponse struct {
	TotalCount    int                       `json:"totalCount" example:"1"` // Number of resource types
	ResourceTypes []FilterCloudResourceType `json:"resourceTypes"`          // A list of resource types
}

type FilterType string

const (
	FilterTypeCloudResourceType FilterType = "cloudResourceType"
	FilterTypeCost              FilterType = "cost"
	FilterTypeInsightMetric     FilterType = "insight-metric"
)

type Filter interface {
	GetFilterID() string
	GetFilterType() FilterType
	GetFilterName() string
}

type FilterCloudResourceType struct {
	FilterType          FilterType  `json:"filterType" example:"cloudResourceType"`
	FilterID            string      `json:"filterId"`
	Connector           source.Type `json:"connector" example:"AWS"`
	ResourceType        string      `json:"resourceType" example:"AWS::AMP::Workspace"`
	ResourceLabel       string      `json:"resourceName" example:""`
	ServiceName         string      `json:"serviceName" example:"amp"`
	ResourceCount       int         `json:"resourceCount" example:"0"`
	ResourceCountChange *float64    `json:"resourceCountChange,omitempty"`
}

func (f FilterCloudResourceType) GetFilterID() string {
	return f.FilterID
}

func (f FilterCloudResourceType) GetFilterType() FilterType {
	return FilterTypeCloudResourceType
}

func (f FilterCloudResourceType) GetFilterName() string {
	return f.ResourceLabel
}

type FilterCost struct {
	FilterType    FilterType  `json:"filterType"`
	FilterID      string      `json:"filterID"`
	CloudProvider source.Type `json:"cloudProvider"`
	ServiceLabel  string      `json:"serviceLabel"`
}

func (f FilterCost) GetFilterID() string {
	return f.FilterID
}

func (f FilterCost) GetFilterType() FilterType {
	return FilterTypeCost
}

func (f FilterCost) GetFilterName() string {
	return f.ServiceLabel
}

type FilterInsightMetric struct {
	FilterType  FilterType  `json:"filterType"`
	FilterID    string      `json:"filterID"`
	InsightID   uint        `json:"insightID"`
	Connector   source.Type `json:"connector"`
	Name        string      `json:"name"`
	Value       int         `json:"value"`
	ValueChange *float64    `json:"valueChange,omitempty"`
}

func (f FilterInsightMetric) GetFilterID() string {
	return f.FilterID
}

func (f FilterInsightMetric) GetFilterType() FilterType {
	return FilterTypeInsightMetric
}

func (f FilterInsightMetric) GetFilterName() string {
	return f.Name
}

type CategoryNode struct {
	CategoryID          string         `json:"categoryID"`
	CategoryName        string         `json:"categoryName"`            // Name of the Category
	ResourceCount       *int           `json:"resourceCount,omitempty"` // Number of Resources of the category
	ResourceCountChange *float64       `json:"resourceCountChange,omitempty"`
	Subcategories       []CategoryNode `json:"subcategories,omitempty"` // Subcategories sorted by ResourceCount [resources/category, ]
	Filters             []Filter       `json:"filters,omitempty"`       // List of Filters associated with this Category
}
