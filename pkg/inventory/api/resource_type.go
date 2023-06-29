package api

import (
	"time"

	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type ResourceType struct {
	Connector     source.Type         `json:"connector" example:"Azure"`                                 // Cloud Provider
	ResourceType  string              `json:"resource_type" example:"Microsoft.Compute/virtualMachines"` // Resource Type
	ResourceLabel string              `json:"resource_name" example:"VM"`                                // Resource Name
	ServiceName   string              `json:"service_name" example:"compute"`                            // Service Name
	Tags          map[string][]string `json:"tags,omitempty"`                                            // Tags
	LogoURI       *string             `json:"logo_uri,omitempty"`                                        // Logo URI

	Count    *int `json:"count" example:"100"`    // Number of Resources of this Resource Type - Metric
	OldCount *int `json:"old_count" example:"90"` // Number of Resources of this Resource Type in the past - Metric

	InsightsCount   *int     `json:"insights_count"`   // Number of Insights that use this Resource Type - Metadata
	ComplianceCount *int     `json:"compliance_count"` // Number of Compliance that use this Resource Type - Metadata
	Insights        []uint   `json:"insights"`         // List of Insights that support this Resource Type - Metadata (GET only)
	Compliance      []string `json:"compliance"`       // List of Compliance that support this Resource Type - Metadata (GET only)
	Attributes      []string `json:"attributes"`       // List supported steampipe Attributes (columns) for this resource type - Metadata (GET only)
}

type ListResourceTypeMetadataResponse struct {
	TotalResourceTypeCount int            `json:"total_resource_type_count" example:"100"`
	ResourceTypes          []ResourceType `json:"resource_types"`
}

type ListResourceTypeMetricsResponse struct {
	TotalCount         int            `json:"total_count"`
	TotalOldCount      int            `json:"total_old_count"`
	TotalResourceTypes int            `json:"total_resource_types"`
	ResourceTypes      []ResourceType `json:"resource_types"`
}

type CountPair struct {
	OldCount int `json:"old_count"`
	Count    int `json:"count"`
}

type ListResourceTypeCompositionResponse struct {
	TotalCount      int                  `json:"total_count"`
	TotalValueCount int                  `json:"total_value_count"`
	TopValues       map[string]CountPair `json:"top_values"`
	Others          CountPair            `json:"others"`
}

type ResourceTypeTrendDatapoint struct {
	Count int       `json:"count"`
	Date  time.Time `json:"date"`
}

type LocationResponse struct {
	Location         string `json:"location"`                              // Region
	ResourceCount    *int   `json:"resourceCount,omitempty" example:"100"` // Number of resources in the region
	ResourceOldCount *int   `json:"resourceOldCount,omitempty" example:"50"`
}

type RegionsResourceCountResponse struct {
	TotalCount int                `json:"totalCount"`
	Regions    []LocationResponse `json:"regions"`
}

type ListRegionsResourceCountCompositionResponse struct {
	TotalCount      int                  `json:"total_count"`
	TotalValueCount int                  `json:"total_value_count"`
	TopValues       map[string]CountPair `json:"top_values"`
	Others          CountPair            `json:"others"`
}
