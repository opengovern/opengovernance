package auth

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
)

type User struct {
	ID         uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	Email      string    `gorm:"not null"`
	ExternalID string    `gorm:"unique;not null"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  sql.NullTime `gorm:"index"`
}

type RoleBinding struct {
	UserID        uuid.UUID `gorm:"uniqueIndex:userid_workspace"`
	WorkspaceName string    `gorm:"uniqueIndex:userid_workspace"`
	Role          api.Role
	AssignedAt    time.Time
}

type Invitation struct {
	ID            uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	Email         string    `gorm:"not null"`
	WorkspaceName string    `gorm:"not null"`
	ExpiredAt     time.Time `gorm:"not null"`
}
