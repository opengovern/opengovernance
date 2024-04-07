package model

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"gorm.io/datatypes"
)

type Credential struct {
	ID                 uuid.UUID      `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()" json:"id"`
	Name               *string        `json:"name,omitempty"`
	ConnectorType      source.Type    `gorm:"not null" json:"connectorType"`
	Secret             string         `json:"-"`
	CredentialType     CredentialType `json:"credentialType"`
	Enabled            bool           `gorm:"default:true" json:"enabled"`
	AutoOnboardEnabled bool           `gorm:"default:false" json:"autoOnboardEnabled"`

	CredentialStoreKeyID      string `json:"credentialStoreKeyId"`
	CredentialStoreKeyVersion string `json:"credentialStoreKeyVersion"`

	LastHealthCheckTime time.Time           `gorm:"not null;default:now()" json:"lastHealthCheckTime"`
	HealthStatus        source.HealthStatus `gorm:"not null;default:'healthy'" json:"healthStatus"`
	HealthReason        *string             `json:"healthReason,omitempty"`
	SpendDiscovery      *bool

	Metadata datatypes.JSON `json:"metadata,omitempty" gorm:"default:'{}'"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt sql.NullTime `gorm:"index"`

	Version int `json:"version"`
}
