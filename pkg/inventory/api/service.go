package api

import (
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type Service struct {
	Connector     source.Type         `json:"connector" example:"Azure"`
	ServiceName   string              `json:"service_name" example:"compute"`
	ServiceLabel  string              `json:"service_label" example:"Compute"`
	ResourceTypes []ResourceType      `json:"resource_types"`
	Tags          map[string][]string `json:"tags,omitempty" swaggertype:"array,string" example:"category:[Data and Analytics,Database,Integration,Management Governance,Storage]"`
	LogoURI       *string             `json:"logo_uri,omitempty" example:"https://kaytu.io/logo.png"`

	ResourceCount    *int `json:"resource_count" example:"100" minimum:"0"`
	OldResourceCount *int `json:"old_resource_count" example:"90" minimum:"0"`
}
