package api

import (
	"time"

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
	Connector source.Type `json:"connector"`
	Name      *string     `json:"name"`
	Config    any         `json:"config"`
}

type Credential struct {
	ID             string                `json:"id"`
	Name           *string               `json:"name,omitempty"`
	ConnectorType  source.Type           `json:"connectorType"`
	CredentialType source.CredentialType `json:"credentialType"`
	Enabled        bool                  `json:"enabled"`
	OnboardDate    time.Time             `json:"onboardDate"`

	LastHealthCheckTime time.Time           `json:"lastHealthCheckTime"`
	HealthStatus        source.HealthStatus `json:"healthStatus"`
	HealthReason        *string             `json:"healthReason,omitempty"`

	Metadata string `json:"metadata,omitempty"`

	Connections []Source `json:"connections,omitempty"`

	TotalConnections     *int `json:"total_connections,omitempty"`
	EnabledConnections   *int `json:"enabled_connections,omitempty"`
	UnhealthyConnections *int `json:"unhealthy_connections,omitempty"`
}
