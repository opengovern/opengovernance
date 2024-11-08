package models

type IntegrationType struct {
	ID               int64  `json:"id"`
	Name             string `json:"name"`
	IntegrationType  string
	Label            string            `json:"label"`
	Tier             string            `json:"tier"`
	Annotations      map[string]string `json:"annotations"`
	Labels           map[string]string `json:"labels"`
	ShortDescription string            `json:"short_description"`
	Description      string            `json:"long_description"`
	Logo             string            `json:"logo"`
	Enabled          bool              `json:"enabled"`
}

type ListIntegrationTypesResponse struct {
	IntegrationTypes []IntegrationType `json:"integration_types"`
	TotalCount       int               `json:"total_count"`
}
