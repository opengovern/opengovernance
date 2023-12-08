package model

import (
	"database/sql"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"time"
)

type Source struct {
	ID           uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"` // Auto-generated UUID
	SourceId     string    `gorm:"index:idx_source_id,unique"`                      // AWS Account ID, Azure Subscription ID, ...
	Name         string    `gorm:"not null"`
	Email        string
	Type         source.Type `gorm:"not null"`
	Description  string
	CredentialID uuid.UUID

	LifecycleState ConnectionLifecycleState `gorm:"not null;default:'enabled'"`

	AssetDiscoveryMethod source.AssetDiscoveryMethodType `gorm:"not null;default:'scheduled'"`

	HealthState         source.HealthStatus
	LastHealthCheckTime time.Time `gorm:"not null;default:now()"`
	HealthReason        *string
	AssetDiscovery      *bool
	SpendDiscovery      *bool

	Connector  Connector  `gorm:"foreignKey:Type;references:Name;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Credential Credential `gorm:"foreignKey:CredentialID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL" json:"-"`

	CreationMethod source.SourceCreationMethod `gorm:"not null;default:'manual'"`

	Metadata datatypes.JSON `gorm:"default:'{}'"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt sql.NullTime `gorm:"index"`
}

// DeleteSource deletes an existing source
func (s *Source) BeforeDelete(tx *gorm.DB) error {
	t := tx.Model(&Source{}).
		Where("id = ?", s.ID.String()).
		Update("lifecycle_state", ConnectionLifecycleStateArchived)
	return t.Error
}

func (s Source) ToAPI() api.Connection {
	metadata := make(map[string]any)
	if s.Metadata.String() != "" {
		_ = json.Unmarshal(s.Metadata, &metadata)
	}
	apiCon := api.Connection{
		ID:                   s.ID,
		ConnectionID:         s.SourceId,
		ConnectionName:       s.Name,
		Email:                s.Email,
		Connector:            s.Type,
		Description:          s.Description,
		CredentialID:         s.CredentialID.String(),
		CredentialName:       s.Credential.Name,
		CredentialType:       s.Credential.CredentialType.ToApi(),
		OnboardDate:          s.CreatedAt,
		HealthState:          s.HealthState,
		LifecycleState:       api.ConnectionLifecycleState(s.LifecycleState),
		AssetDiscoveryMethod: s.AssetDiscoveryMethod,
		LastHealthCheckTime:  s.LastHealthCheckTime,
		HealthReason:         s.HealthReason,
		Metadata:             metadata,
		AssetDiscovery:       s.AssetDiscovery,
		SpendDiscovery:       s.SpendDiscovery,

		ResourceCount: nil,
		Cost:          nil,
		LastInventory: nil,
	}
	return apiCon
}
