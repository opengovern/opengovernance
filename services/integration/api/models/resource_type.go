package models

import "github.com/opengovern/og-util/pkg/integration"

type ResourceTypeConfiguration struct {
	Name            string           `json:"name"`
	IntegrationType integration.Type `json:"integration_type"`
	Description     string           `json:"description"`
	Params          []Param          `json:"params"`
}

type Param struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Required    bool    `json:"required"`
	Default     *string `json:"default"`
}

type ListIntegrationTypeResourceTypesResponse struct {
	ResourceTypes []ResourceTypeConfiguration `json:"integration_types"`
	TotalCount    int                         `json:"total_count"`
}
