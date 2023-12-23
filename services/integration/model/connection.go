package model

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Connection struct {
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
func (s *Connection) BeforeDelete(tx *gorm.DB) error {
	t := tx.Model(new(Connection)).
		Where("id = ?", s.ID.String()).
		Update("lifecycle_state", ConnectionLifecycleStateArchived)
	return t.Error
}

func (s *Connection) TableName() string {
	return "sources"
}
