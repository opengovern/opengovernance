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

type IntegrationTypeIntegrationCount struct {
	Total    int64 `json:"total"`
	Active   int64 `json:"active"`
	Inactive int64 `json:"inactive"`
	Archived int64 `json:"archived"`
	Demo     int64 `json:"demo"`
}

type ListIntegrationTypesItem struct {
	ID           int64                           `json:"id"`
	Name         string                          `json:"name"`
	PlatformName string                          `json:"platform_name"`
	Tier         string                          `json:"tier"`
	Title        string                          `json:"title"`
	Logo         string                          `json:"logo"`
	Enabled      bool                            `json:"enabled"`
	Count        IntegrationTypeIntegrationCount `json:"count"`
}

type ListIntegrationTypesResponse struct {
	IntegrationTypes []ListIntegrationTypesItem `json:"integration_types"`
	TotalCount       int                        `json:"total_count"`
}
