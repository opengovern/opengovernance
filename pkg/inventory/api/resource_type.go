package api

import (
	"time"

	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type ResourceType struct {
	Connector     source.Type         `json:"connector"`
	ResourceType  string              `json:"resource_type"`
	ResourceLabel string              `json:"resource_name"`
	ServiceName   string              `json:"service_name"`
	Tags          map[string][]string `json:"tags,omitempty"`
	LogoURI       *string             `json:"logo_uri,omitempty"`

	Count              *int     `json:"count,omitempty"`                // Number of Resources of this Resource Type - Metric
	CountChangePercent *float64 `json:"count_change_percent,omitempty"` // Percentage change in the number of Resources of this Resource Type - Metric

	InsightsCount   *int     `json:"insights_count,omitempty"`   // Number of Insights that use this Resource Type - Metadata
	ComplianceCount *int     `json:"compliance_count,omitempty"` // Number of Compliance that use this Resource Type - Metadata
	Insights        []uint   `json:"insights,omitempty"`         // List of Insights that support this Resource Type - Metadata (GET only)
	Compliance      []string `json:"compliance,omitempty"`       // List of Compliance that support this Resource Type - Metadata (GET only)
	Attributes      []string `json:"attributes,omitempty"`       // List supported steampipe Attributes (columns) for this resource type - Metadata (GET only)
}

type ListResourceTypeMetadataResponse struct {
	TotalResourceTypeCount int            `json:"total_resource_type_count"`
	ResourceTypes          []ResourceType `json:"resource_types"`
}

type ListResourceTypeMetricsResponse struct {
	TotalCount         int            `json:"total_count"`
	TotalResourceTypes int            `json:"total_resource_types"`
	ResourceTypes      []ResourceType `json:"resource_types"`
}

type ListResourceTypeCompositionResponse struct {
	TotalCount      int            `json:"total_count"`
	TotalValueCount int            `json:"total_value_count"`
	TopValues       map[string]int `json:"top_values"`
	Others          int            `json:"others"`
}

type ResourceTypeTrendDatapoint struct {
	Count int       `json:"count"`
	Date  time.Time `json:"date"`
}
