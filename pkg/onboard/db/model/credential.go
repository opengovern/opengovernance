package model

import (
	"database/sql"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"gorm.io/datatypes"
	"time"
)

type Credential struct {
	ID                 uuid.UUID      `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()" json:"id"`
	Name               *string        `json:"name,omitempty"`
	ConnectorType      source.Type    `gorm:"not null" json:"connectorType"`
	Secret             string         `json:"-"`
	CredentialType     CredentialType `json:"credentialType"`
	Enabled            bool           `gorm:"default:true" json:"enabled"`
	AutoOnboardEnabled bool           `gorm:"default:false" json:"autoOnboardEnabled"`

	LastHealthCheckTime time.Time           `gorm:"not null;default:now()" json:"lastHealthCheckTime"`
	HealthStatus        source.HealthStatus `gorm:"not null;default:'healthy'" json:"healthStatus"`
	HealthReason        *string             `json:"healthReason,omitempty"`

	Metadata datatypes.JSON `json:"metadata,omitempty" gorm:"default:'{}'"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt sql.NullTime `gorm:"index"`

	Version int `json:"version"`
}

func (credential *Credential) ToAPI() api.Credential {
	metadata := make(map[string]any)
	if string(credential.Metadata) == "" {
		credential.Metadata = []byte("{}")
	}
	_ = json.Unmarshal(credential.Metadata, &metadata)
	apiCredential := api.Credential{
		ID:                  credential.ID.String(),
		Name:                credential.Name,
		ConnectorType:       credential.ConnectorType,
		CredentialType:      credential.CredentialType.ToApi(),
		Enabled:             credential.Enabled,
		AutoOnboardEnabled:  credential.AutoOnboardEnabled,
		OnboardDate:         credential.CreatedAt,
		LastHealthCheckTime: credential.LastHealthCheckTime,
		HealthStatus:        credential.HealthStatus,
		HealthReason:        credential.HealthReason,
		Metadata:            metadata,

		Config: "",

		Connections:           nil,
		TotalConnections:      nil,
		OnboardConnections:    nil,
		UnhealthyConnections:  nil,
		DiscoveredConnections: nil,
	}

	return apiCredential
}
