package models

type CreateRequest struct {
	IntegrationType string         `json:"integration_type"`
	CredentialType  string         `json:"credential_type"`
	Credentials     map[string]any `json:"credentials"`
}

type UpdateRequest struct {
	Config map[string]any `json:"config"`
}

type CredentialItem struct {
	ID             string `json:"id"`
	CredentialType string `json:"type"`
}

type ListResponse struct {
	Credentials []CredentialItem `json:"credentials"`
	TotalCount  int              `json:"total_count"`
}
