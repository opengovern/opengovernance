package api

import (
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/describe/es"

	"gitlab.com/keibiengine/keibi-engine/pkg/source"

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
	SortFieldName          SortFieldType = "resourceName"
	SortFieldSourceType    SortFieldType = "provider"
	SortFieldResourceType  SortFieldType = "resourceType"
	SortFieldResourceGroup SortFieldType = "resourceGroup"
	SortFieldLocation      SortFieldType = "location"
	SortFieldSourceID      SortFieldType = "connectionID"
)

type GetResourceRequest struct {
	ResourceType string `json:"resourceType" validate:"required"`
	ID           string `json:"ID" validate:"required"` //	Resource ID
}

type LocationByProviderResponse struct {
	Name string `json:"name"`
}

type RunQueryRequest struct {
	Page Page `json:"page" validate:"required"`
	// NOTE: we don't support multi-field sort for now, if sort is empty, results would be sorted by first column
	Sorts []SmartQuerySortItem `json:"sorts"`
}

type RunQueryResponse struct {
	Title   string   `json:"title"`
	Query   string   `json:"query"`
	Headers []string `json:"headers"` // column names
	// result of query. in order to access a specific cell please use Result[Row][Column]
	Result [][]interface{} `json:"result"`
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
// @Description if you provide two values for same filter OR operation would be used
// @Description if you provide value for two filters AND operation would be used
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
// @Description if you provide two values for same filter OR operation would be used
// @Description if you provide value for two filters AND operation would be used
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
	Category []string `json:"category"`
	// if you dont need to use this filter, leave them empty. (e.g. [])
	Service []string `json:"service"`
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
	Resources  []AllResource `json:"resources"`
	TotalCount int64         `json:"totalCount,omitempty"`
}

type AllResource struct {
	ResourceName           string     `json:"resourceName"`
	ResourceID             string     `json:"resourceID"`
	ResourceType           string     `json:"resourceType"`
	ResourceTypeName       string     `json:"resourceTypeName"`
	ResourceCategory       string     `json:"resourceCategory"`
	Provider               SourceType `json:"provider"`
	Location               string     `json:"location"`
	ProviderConnectionID   string     `json:"providerConnectionID"`
	ProviderConnectionName string     `json:"providerConnectionName"`

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
	Resources  []AzureResource `json:"resources"`
	TotalCount int64           `json:"totalCount,omitempty"`
}

type AzureResource struct {
	ResourceName           string `json:"resourceName"`
	ResourceID             string `json:"resourceID"`
	ResourceType           string `json:"resourceType"`
	ResourceTypeName       string `json:"resourceTypeName"`
	ResourceCategory       string `json:"resourceCategory"`
	ResourceGroup          string `json:"resourceGroup"`
	Location               string `json:"location"`
	ProviderConnectionID   string `json:"providerConnectionID"`
	ProviderConnectionName string `json:"providerConnectionName"`

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
	Resources  []AWSResource `json:"resources"`
	TotalCount int64         `json:"totalCount,omitempty"`
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
	ID          uint              `json:"id"`
	Provider    string            `json:"provider"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Category    string            `json:"category"`
	Query       string            `json:"query"`
	Tags        map[string]string `json:"tags"`
}

type TrendDataPoint struct {
	Timestamp int64 `json:"timestamp"`
	Value     int64 `json:"value"`
}

type FloatTrendDataPoint struct {
	Timestamp int64   `json:"timestamp"`
	Value     float64 `json:"value"`
}

type CategoryResourceTrend struct {
	Name  string           `json:"name"`
	Trend []TrendDataPoint `json:"trend"`
}

type CategoryCostTrend struct {
	Name  string                `json:"name"`
	Trend []FloatTrendDataPoint `json:"trend"`
}

type ResourceGrowthTrendResponse struct {
	CategoryName  string                  `json:"categoryName"`
	Trend         []TrendDataPoint        `json:"trend"`
	Subcategories []CategoryResourceTrend `json:"Subcategories"`
}

type CostGrowthTrendResponse struct {
	CategoryName  string                `json:"categoryName"`
	Trend         []FloatTrendDataPoint `json:"trend"`
	Subcategories []CategoryCostTrend   `json:"Subcategories"`
}

type ListQueryRequest struct {
	TitleFilter    string      `json:"titleFilter"`
	ProviderFilter *SourceType `json:"providerFilter"`
	Labels         []string    `json:"labels"`
}

type AccountResourceCountResponse struct {
	SourceID               string      `json:"sourceID"`
	SourceType             source.Type `json:"sourceType"`
	ProviderConnectionName string      `json:"providerConnectionName"`
	ProviderConnectionID   string      `json:"providerConnectionID"`
	Enabled                bool        `json:"enabled"`
	ResourceCount          int         `json:"resourceCount"`
	OnboardDate            time.Time   `json:"onboardDate"`
	LastInventory          time.Time   `json:"lastInventory"`
}

type TopAccountResponse struct {
	SourceID               string `json:"sourceID"`
	Provider               string `json:"provider"`
	ProviderConnectionName string `json:"providerConnectionName"`
	ProviderConnectionID   string `json:"providerConnectionID"`
	ResourceCount          int    `json:"resourceCount"`
}

type TopAccountCostResponse struct {
	SourceID               string  `json:"sourceID"`
	ProviderConnectionName string  `json:"providerConnectionName"`
	ProviderConnectionID   string  `json:"providerConnectionID"`
	Cost                   float64 `json:"cost"`
}

type TopServicesResponse struct {
	ServiceName      string `json:"serviceName"`
	Provider         string `json:"provider"`
	ResourceCount    int    `json:"resourceCount"`
	LastDayCount     *int   `json:"lastDayCount"`
	LastWeekCount    *int   `json:"lastWeekCount"`
	LastQuarterCount *int   `json:"lastQuarterCount"`
	LastYearCount    *int   `json:"lastYearCount"`
}

type TopServiceCostResponse struct {
	ServiceName string  `json:"serviceName"`
	Cost        float64 `json:"cost"`
}

type ResourceTypeResponse struct {
	ResourceType     string `json:"resourceType"`
	ResourceTypeName string `json:"resourceTypeName"`
	ResourceCount    int    `json:"resourceCount"`
	LastDayCount     *int   `json:"lastDayCount"`
	LastWeekCount    *int   `json:"lastWeekCount"`
	LastQuarterCount *int   `json:"lastQuarterCount"`
	LastYearCount    *int   `json:"lastYearCount"`
}

type CategorizedMetricsResponse struct {
	Category map[string][]ResourceTypeResponse `json:"category"`
}

type LocationResponse struct {
	Location      string `json:"location"`
	ResourceCount *int   `json:"resourceCount,omitempty"`
}

type Filter interface {
}

type FilterCloudResourceType struct {
	FilterID      string       `json:"filterId"`
	CloudProvider source.Type  `json:"cloudProvider"`
	ResourceType  string       `json:"resourceType"`
	ResourceName  string       `json:"resourceName"`
	ResourceCount HistoryCount `json:"resourceCount"`
}

type Cost struct {
	CurrentCost float64 `json:"current_cost"`
	HistoryCost float64 `json:"history_cost"`
}

type FilterCost struct {
	FilterID      string      `json:"filterID"`
	CloudProvider source.Type `json:"cloudProvider"`
	Cost          Cost        `json:"cost"`
}

type CategoryNode struct {
	CategoryID    string         `json:"categoryID"`
	CategoryName  string         `json:"categoryName"`
	ResourceCount *HistoryCount  `json:"resourceCount,omitempty"`
	Cost          *Cost          `json:"cost,omitempty"`
	Subcategories []CategoryNode `json:"subcategories,omitempty"`
	Filters       []Filter       `json:"filters,omitempty"`
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
	ServiceName  string         `json:"serviceName"`
	Distribution map[string]int `json:"distribution"`
}

type Category struct {
	Name        string   `json:"name"`
	SubCategory []string `json:"subCategory"`
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
	Categories    map[string]ConnectionSummaryCategory `json:"categories"`
	CloudServices map[string]int                       `json:"cloudServices"`
	ResourceTypes map[string]int                       `json:"resourceTypes"`
}
