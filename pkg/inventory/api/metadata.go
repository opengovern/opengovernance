package api

import "gitlab.com/keibiengine/keibi-engine/pkg/source"

type ConnectorMetadata struct {
	Connector      source.Type `json:"connector"`       // Connector
	ConnectorLabel string      `json:"connector_label"` // Connector Lable
	Services       []string    `json:"services"`        // List of cloud services
	ResourceTypes  []string    `json:"resource_types"`  // List of resource types
}

type ServiceMetadata struct {
	Connector     source.Type `json:"connector"`                // Service Connector
	ServiceName   string      `json:"service_name"`             // Service Name
	ServiceLabel  string      `json:"service_label"`            // Service Lable
	ParentService *string     `json:"parent_service,omitempty"` // Parent service
	ResourceTypes []string    `json:"resource_types"`           // List of resource types

	CostSupport         bool     `json:"cost_support"`                     // Cost is supported [yes/no]
	CostMapServiceNames []string `json:"cost_map_service_names,omitempty"` // List of Cost map service names
}

type ResourceTypeMetadata struct {
	Connector         source.Type `json:"connector"`           // Resource type connector
	ResourceTypeName  string      `json:"resource_type_name"`  // Resource type name
	ResourceTypeLabel string      `json:"resource_type_label"` // Resource type lable
	ServiceName       string      `json:"service_name"`        // Platform Patern Service name
	DiscoveryEnabled  bool        `json:"discovery_enabled"`   // Discovery support enabled

	Insights   []uint   `json:"insights,omitempty"`   // List of Insights that support this Resource Type
	Compliance []string `json:"compliance,omitempty"` // List of Compliances that support this Resource Type
}
