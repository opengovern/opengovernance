package api

import (
	"time"

	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type CreateCredentialRequest struct {
	SourceType source.Type `json:"source_type" example:"Azure"`
	Config     any         `json:"config"`
}

type CreateCredentialResponse struct {
	ID string `json:"id"`
}

type UpdateCredentialRequest struct {
	Connector source.Type `json:"connector" example:"Azure"`
	Name      *string     `json:"name"`
	Config    any         `json:"config"`
}

type ListCredentialResponse struct {
	TotalCredentialCount int          `json:"totalCredentialCount" example:"5" minimum:"0"`
	Credentials          []Credential `json:"credentials"`
}

type CredentialType string

const (
	CredentialTypeAutoAzure             CredentialType = "auto-azure"
	CredentialTypeAutoAws               CredentialType = "auto-aws"
	CredentialTypeManualAwsOrganization CredentialType = "manual-aws-org"
	CredentialTypeManualAzureSpn        CredentialType = "manual-azure-spn"
)

type Credential struct {
	ID                 string         `json:"id" example:"1028642a-b22e-26ha-c5h2-22nl254678m5"`
	Name               *string        `json:"name,omitempty" example:"a-1mahsl7lzk"`
	ConnectorType      source.Type    `json:"connectorType" example:"AWS"`
	CredentialType     CredentialType `json:"credentialType" example:"manual-aws-org"`
	Enabled            bool           `json:"enabled" example:"true"`
	AutoOnboardEnabled bool           `json:"autoOnboardEnabled" example:"false"`
	OnboardDate        time.Time      `json:"onboardDate" format:"date-time" example:"2023-06-03T12:21:33.406928Z"`

	Config any `json:"config"`

	LastHealthCheckTime time.Time           `json:"lastHealthCheckTime" format:"date-time" example:"2023-06-03T12:21:33.406928Z"`
	HealthStatus        source.HealthStatus `json:"healthStatus" example:"healthy"`
	HealthReason        *string             `json:"healthReason,omitempty" example:""`

	Metadata map[string]any `json:"metadata,omitempty" swaggertype:"object"`

	Connections []Connection `json:"connections,omitempty"`

	TotalConnections     *int `json:"total_connections" example:"300" minimum:"0"`
	EnabledConnections   *int `json:"enabled_connections" example:"250" minimum:"0"`
	UnhealthyConnections *int `json:"unhealthy_connections" example:"50" minimum:"0"`
}
