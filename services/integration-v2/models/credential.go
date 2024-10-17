package models

import (
	"database/sql"
	"github.com/google/uuid"
	"time"
)

type Credential struct {
	ID             uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()" json:"id"`
	Secret         string    `json:"-"`
	CredentialType string    `json:"credentialType"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt sql.NullTime `gorm:"index"`
}
