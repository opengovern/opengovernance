package models

import (
	"github.com/opengovern/og-util/pkg/integration"
	"time"
)

type Credential struct {
	ID              string            `json:"id"`
	Secret          string            `json:"secret"`
	IntegrationType integration.Type  `json:"integration_type"`
	CredentialType  string            `json:"credential_type"`
	Metadata        map[string]string `json:"metadata"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

type ListCredentialsRequest struct {
	CredentialID    []string `json:"credential_id"`
	IntegrationType []string `json:"integration_type"`
}

type UpdateCredentialRequest struct {
	Credentials map[string]any `json:"credentials"`
}

type ListCredentialsResponse struct {
	Credentials []Credential `json:"credentials"`
	TotalCount  int          `json:"total_count"`
}
