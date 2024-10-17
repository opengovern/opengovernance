package models

import (
	"database/sql"
	"github.com/google/uuid"
	"github.com/jackc/pgtype"
	"time"
)

type Integration struct {
	IntegrationTracker uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"` // Auto-generated UUID
	IntegrationID      string
	IntegrationName    string
	Connector          string
	Type               string
	OnboardDate        time.Time
	Metadata           pgtype.JSONB

	CredentialID uuid.UUID

	Lifecycle string
	Reason    string
	LastCheck *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt sql.NullTime `gorm:"index"`
}
