package api

import (
	"time"

	"github.com/kaytu-io/kaytu-engine/pkg/describe/es"
	"github.com/kaytu-io/kaytu-util/pkg/source"

	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
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

type Page struct {
	No   int `json:"no,omitempty"`
	Size int `json:"size,omitempty"`
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

type ResourceSortItem struct {
	Field     SortFieldType `json:"field" enums:"resourceID,connector,resourceType,resourceGroup,location,connectionID"`
	Direction DirectionType `json:"direction" enums:"asc,desc"`
}

type SmartQuerySortItem struct {
	// fill this with column name
	Field     string        `json:"field"`
	Direction DirectionType `json:"direction" enums:"asc,desc"`
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
	Total kaytu.SearchTotal `json:"total"`
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

type ConnectionData struct {
	ConnectionID         string     `json:"connectionID"`
	Count                *int       `json:"count"`
	OldCount             *int       `json:"oldCount"`
	LastInventory        *time.Time `json:"lastInventory"`
	TotalCost            *float64   `json:"cost"`
	DailyCostAtStartTime *float64   `json:"dailyCostAtStartTime"`
	DailyCostAtEndTime   *float64   `json:"dailyCostAtEndTime"`
}
