package api

import "gitlab.com/keibiengine/keibi-engine/pkg/source"

type ConnectorMetadata struct {
	Connector          source.Type `json:"connector"`
	ConnectorLabel     string      `json:"connector_label"`
	ServicesCount      *int        `json:"services_count,omitempty"`
	Services           []string    `json:"services,omitempty"`
	ResourceTypesCount *int        `json:"resource_types_count,omitempty"`
	ResourceTypes      []string    `json:"resource_types,omitempty"`
	LogoURI            *string     `json:"logo_uri,omitempty"`
}

type ServiceMetadata struct {
	Connector          source.Type `json:"connector"`
	ServiceName        string      `json:"service_name"`
	ServiceLabel       string      `json:"service_label"`
	ParentService      *string     `json:"parent_service,omitempty"`
	ResourceTypesCount *int        `json:"resource_types_count,omitempty"`
	ResourceTypes      []string    `json:"resource_types,omitempty"`
	LogoURI            *string     `json:"logo_uri,omitempty"`

	CostSupport         bool     `json:"cost_support"`
	CostMapServiceNames []string `json:"cost_map_service_names,omitempty"`
}

type ResourceTypeMetadata struct {
	Connector         source.Type `json:"connector"`
	ResourceTypeName  string      `json:"resource_type_name"`
	ResourceTypeLabel string      `json:"resource_type_label"`
	ServiceName       string      `json:"service_name"`
	DiscoveryEnabled  bool        `json:"discovery_enabled"`
	LogoURI           *string     `json:"logo_uri,omitempty"`

	InsightsCount   *int `json:"insights_count,omitempty"`
	ComplianceCount *int `json:"compliance_count,omitempty"`

	Attributes []string `json:"attributes,omitempty"`
	Insights   []uint   `json:"insights,omitempty"`
	Compliance []string `json:"compliance,omitempty"`
}
