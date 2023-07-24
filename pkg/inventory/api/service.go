package api

import (
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type Service struct {
	Connector     source.Type         `json:"connector" example:"Azure"`
	ServiceName   string              `json:"service_name" example:"compute"`
	ServiceLabel  string              `json:"service_label" example:"Compute"`
	ResourceTypes []ResourceType      `json:"resource_types"`
	Tags          map[string][]string `json:"tags,omitempty"`
	LogoURI       *string             `json:"logo_uri,omitempty" example:"https://kaytu.io/logo.png"`

	ResourceCount    *int `json:"resource_count" example:"100" minimum:"0"`
	OldResourceCount *int `json:"old_resource_count" example:"90" minimum:"0"`
}

type ListServiceMetricsResponse struct {
	TotalCount    int       `json:"total_count" example:"10000" minimum:"0"`
	TotalOldCount int       `json:"total_old_count" example:"9000" minimum:"0"`
	TotalServices int       `json:"total_services" example:"50" minimum:"0"`
	Services      []Service `json:"services"`
}

type ListServiceMetadataResponse struct {
	TotalServiceCount int       `json:"total_service_count" example:"100" minimum:"0"`
	Services          []Service `json:"services"`
}
