package api

import (
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
)

type CreateCredentialRequest struct {
	Name       string      `json:"name"`
	SourceType source.Type `json:"source_type"`
	Config     any         `json:"config"`
}

type CreateCredentialResponse struct {
	ID string `json:"id"`
}

type UpdateCredentialRequest struct {
	ID         string      `json:"id"`
	SourceType source.Type `json:"source_type"`
	Name       *string     `json:"name"`
	Config     any         `json:"config"`
}

type Credential struct {
	ID             string                  `json:"id"`
	Name           *string                 `json:"name,omitempty"`
	ConnectorType  source.Type             `json:"connectorType"`
	Status         source.CredentialStatus `json:"status"`
	CredentialType source.CredentialType   `json:"credentialType"`

	LastHealthCheckTime int64               `json:"lastHealthCheckTime"`
	HealthStatus        source.HealthStatus `json:"healthStatus"`
	HealthReason        *string             `json:"healthReason,omitempty"`

	Metadata string `json:"metadata,omitempty"`

	Connections []Source `json:"connections,omitempty"`
}
