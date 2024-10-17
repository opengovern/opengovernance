package models

type CreateRequest struct {
	CredentialType string         `json:"type"`
	Config         map[string]any `json:"config"`
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
