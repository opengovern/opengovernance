package api

import "gitlab.com/keibiengine/keibi-engine/pkg/source"

type ConnectorMetadata struct {
	Connector      source.Type `json:"connector"`
	ConnectorLabel string      `json:"connector_label"`
	Services       []string    `json:"services"`
	ResourceTypes  []string    `json:"resource_types"`
}

type ServiceMetadata struct {
	Connector     source.Type `json:"connector"`
	ServiceName   string      `json:"service_name"`
	ServiceLabel  string      `json:"service_label"`
	ParentService *string     `json:"parent_service,omitempty"`
	ResourceTypes []string    `json:"resource_types"`

	CostSupport         bool     `json:"cost_support"`
	CostMapServiceNames []string `json:"cost_map_service_names,omitempty"`
}

type ResourceTypeMetadata struct {
	Connector         source.Type `json:"connector"`
	ResourceTypeName  string      `json:"resource_type_name"`
	ResourceTypeLabel string      `json:"resource_type_label"`
	ServiceName       string      `json:"service_name"`
	DiscoveryEnabled  bool        `json:"discovery_enabled"`

	Insights   []uint   `json:"insights,omitempty"`
	Compliance []string `json:"compliance,omitempty"`
}
