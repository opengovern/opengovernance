package models

type IntegrationType struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	PlatformName string `json:"platform_name"`
	Label        string `json:"label"`
	Tier         string `json:"tier"`
	Logo         string `json:"logo"`
	Enabled      bool   `json:"enabled"`
}

type ListIntegrationTypesResponse struct {
	IntegrationTypes []IntegrationType `json:"integration_types"`
	TotalCount       int               `json:"total_count"`
}
