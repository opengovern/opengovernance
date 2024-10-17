package models

import (
	"database/sql"
	"github.com/google/uuid"
	"time"
)

type Credential struct {
	ID              uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"` // Auto-generated UUID
	CredentialType  string
	HealthStatus    string
	LastHealthCheck *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt sql.NullTime `gorm:"index"`
}
