package api

import (
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type Service struct {
	Connector     source.Type         `json:"connector"`
	ServiceName   string              `json:"service_name"`
	ServiceLabel  string              `json:"service_label"`
	ResourceTypes []ResourceType      `json:"resource_types"`
	Tags          map[string][]string `json:"tags,omitempty"`
	LogoURI       *string             `json:"logo_uri,omitempty"`

	ResourceCount    *int `json:"resource_count"`
	OldResourceCount *int `json:"old_resource_count"`
}

type ListServiceMetricsResponse struct {
	TotalCount    int       `json:"total_count"`
	TotalServices int       `json:"total_services"`
	Services      []Service `json:"services"`
}

type ListServiceMetadataResponse struct {
	TotalServiceCount int       `json:"total_service_count"`
	Services          []Service `json:"services"`
}
