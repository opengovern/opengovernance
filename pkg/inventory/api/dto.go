package api

import (
	"time"

	"github.com/kaytu-io/kaytu-engine/pkg/describe/es"
	"github.com/kaytu-io/kaytu-util/pkg/source"

	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
)

type DirectionType string

const (
	DirectionAscending  DirectionType = "asc"
	DirectionDescending DirectionType = "desc"
)

type SortFieldType string

const (
	SortFieldResourceID    SortFieldType = "resourceID"
	SortFieldConnector     SortFieldType = "connector"
	SortFieldResourceType  SortFieldType = "resourceType"
	SortFieldResourceGroup SortFieldType = "resourceGroup"
	SortFieldLocation      SortFieldType = "location"
	SortFieldConnectionID  SortFieldType = "connectionID"
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
	Service []string `json:"service"`
	// if you dont need to use this filter, leave them empty. (e.g. [])
	Location []string `json:"location"`
	// if you dont need to use this filter, leave them empty. (e.g. [])
	ConnectionID []string `json:"connectionID"`
	// if you dont need to use this filter, leave them empty. (e.g. [])
	Connectors []source.Type `json:"connectors"`
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
	Field     SortFieldType `json:"field" enums:"resourceID,connector,resourceType,resourceGroup,location,connectionID"`
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
	ResourceName           string      `json:"resourceName"`           // Resource Name
	ResourceID             string      `json:"resourceID"`             // Resource Id
	ResourceType           string      `json:"resourceType"`           // Resource Type
	ResourceTypeLabel      string      `json:"resourceTypeLabel"`      // Resource Type Label
	Connector              source.Type `json:"connector"`              // Resource Provider
	Location               string      `json:"location"`               // The Region of the resource
	ConnectionID           string      `json:"connectionID"`           // Kaytu Connection Id of the resource
	ProviderConnectionID   string      `json:"providerConnectionID"`   // Provider Connection Id
	ProviderConnectionName string      `json:"providerConnectionName"` // Provider Connection Name

	Attributes map[string]string `json:"attributes"`
}

type GetAzureResourceResponse struct {
	Resources  []AzureResource `json:"resources"`            // A list of Azure resources with details
	TotalCount int64           `json:"totalCount,omitempty"` // Number of returned resources
}

type AzureResource struct {
	ResourceName           string `json:"resourceName"`           // Resource Name
	ResourceID             string `json:"resourceID"`             // Resource Id
	ResourceType           string `json:"resourceType"`           // Resource Type
	ResourceTypeLabel      string `json:"resourceTypeLabel"`      // Resource Type Label
	ResourceGroup          string `json:"resourceGroup"`          // Resource Group
	Location               string `json:"location"`               // The Region of the resource
	ConnectionID           string `json:"connectionID"`           // Kaytu Connection Id of the resource
	ProviderConnectionID   string `json:"providerConnectionID"`   // Provider Connection Id
	ProviderConnectionName string `json:"providerConnectionName"` // Provider Connection Name

	Attributes map[string]string `json:"attributes"`
}

type AWSResource struct {
	ResourceName           string `json:"resourceName"`
	ResourceID             string `json:"resourceID"`
	ResourceType           string `json:"resourceType"`
	ResourceTypeLabel      string `json:"resourceTypeLabel"`
	Location               string `json:"location"`
	ConnectionID           string `json:"connectionID"`
	ProviderConnectionID   string `json:"ProviderConnectionID"`
	ProviderConnectionName string `json:"providerConnectionName"`

	Attributes map[string]string `json:"attributes"`
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
	Sort    []any             `json:"sort"`
}

type GenericQueryResponse struct {
	Hits GenericQueryHits `json:"hits"`
}
type GenericQueryHits struct {
	Total keibi.SearchTotal `json:"total"`
	Hits  []GenericQueryHit `json:"hits"`
}
type GenericQueryHit struct {
	ID      string         `json:"_id"`
	Score   float64        `json:"_score"`
	Index   string         `json:"_index"`
	Type    string         `json:"_type"`
	Version int64          `json:"_version,omitempty"`
	Source  map[string]any `json:"_source"`
	Sort    []any          `json:"sort"`
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
	TitleFilter string        `json:"titleFilter"`      // Specifies the Title
	Connectors  []source.Type `json:"connectorsFilter"` // Specifies the Connectors
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
	ConnectionID         string     `json:"connectionID"`
	Count                *int       `json:"count"`
	OldCount             *int       `json:"oldCount"`
	LastInventory        *time.Time `json:"lastInventory"`
	TotalCost            *float64   `json:"cost"`
	DailyCostAtStartTime *float64   `json:"dailyCostAtStartTime"`
	DailyCostAtEndTime   *float64   `json:"dailyCostAtEndTime"`
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
