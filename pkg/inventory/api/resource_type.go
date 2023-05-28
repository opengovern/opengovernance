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
	Count         *int                `json:"count,omitempty"`
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
