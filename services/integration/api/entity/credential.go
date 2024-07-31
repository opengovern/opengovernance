package entity

import (
	"encoding/json"
	"time"

	"github.com/kaytu-io/kaytu-engine/services/integration/model"
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type AWSCredentialConfig struct {
	AccountID      string  `json:"accountID"`
	AssumeRoleName string  `json:"assumeRoleName"`
	ExternalId     *string `json:"externalId,omitempty"`

	AccessKey *string `json:"accessKey,omitempty"`
	SecretKey *string `json:"secretKey,omitempty"`
}

func (s AWSCredentialConfig) AsMap() map[string]any {
	in, err := json.Marshal(s)
	if err != nil {
		panic(err) // Don't expect any error
	}

	var out map[string]any
	if err := json.Unmarshal(in, &out); err != nil {
		panic(err) // Don't expect any error
	}

	return out
}

type CreateAWSCredentialRequest struct {
	Config AWSCredentialConfig `json:"config,omitempty"`
}

type CreateCredentialResponse struct {
	ID          string       `json:"id"`
	Connections []Connection `json:"connections"`
}

type UpdateAWSCredentialRequest struct {
	Name   *string              `json:"name"`
	Config *AWSCredentialConfig `json:"config"`
}

type UpdateAzureCredentialRequest struct {
	Name   *string                `json:"name"`
	Config *AzureCredentialConfig `json:"config"`
}

type ListCredentialResponse struct {
	TotalCredentialCount int          `json:"totalCredentialCount" example:"5" minimum:"0" maximum:"20"`
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

	Config  any `json:"config"`
	Version int `json:"version"`

	LastHealthCheckTime time.Time           `json:"lastHealthCheckTime" format:"date-time" example:"2023-06-03T12:21:33.406928Z"`
	HealthStatus        source.HealthStatus `json:"healthStatus" example:"healthy"`
	HealthReason        *string             `json:"healthReason,omitempty" example:""`
	SpendDiscovery      *bool               `json:"spendDiscovery"`

	Metadata map[string]any `json:"metadata,omitempty"`

	Connections []Connection `json:"connections,omitempty"`

	TotalConnections     *int64 `json:"total_connections" example:"300" minimum:"0" maximum:"1000"`
	UnhealthyConnections *int64 `json:"unhealthy_connections" example:"50" minimum:"0" maximum:"100"`

	DiscoveredConnections *int64 `json:"discovered_connections" example:"50" minimum:"0" maximum:"100"`
	OnboardConnections    *int64 `json:"onboard_connections" example:"250" minimum:"0" maximum:"1000"`
	DisabledConnections   *int64 `json:"disabled_connections" example:"0" minimum:"0" maximum:"1000"`
	ArchivedConnections   *int64 `json:"archived_connections" example:"0" minimum:"0" maximum:"1000"`
}

// NewCredential creates API compatible credential from model credential.
func NewCredential(credential model.Credential) Credential {
	metadata := make(map[string]any)
	if string(credential.Metadata) == "" {
		credential.Metadata = []byte("{}")
	}
	_ = json.Unmarshal(credential.Metadata, &metadata)
	apiCredential := Credential{
		ID:                  credential.ID.String(),
		Name:                credential.Name,
		ConnectorType:       credential.ConnectorType,
		CredentialType:      NewCredentialType(credential.CredentialType),
		Enabled:             credential.Enabled,
		AutoOnboardEnabled:  credential.AutoOnboardEnabled,
		OnboardDate:         credential.CreatedAt,
		LastHealthCheckTime: credential.LastHealthCheckTime,
		HealthStatus:        credential.HealthStatus,
		HealthReason:        credential.HealthReason,
		Metadata:            metadata,
		Version:             credential.Version,
		SpendDiscovery:      credential.SpendDiscovery,

		Config: "",

		Connections:           nil,
		TotalConnections:      nil,
		OnboardConnections:    nil,
		UnhealthyConnections:  nil,
		DiscoveredConnections: nil,
	}

	return apiCredential
}

func NewCredentialType(c model.CredentialType) CredentialType {
	return CredentialType(c)
}
