package configs

type IntegrationCredentials struct {
	APIKey         string `json:"api_key"`
	ProjectID      string `json:"project_id"`
	ProjectName    string `json:"project_name"`
	OrganizationID string `json:"organization_id"`
}
