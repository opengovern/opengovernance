package auth

import (
	"time"

	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model

	ID         uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	Email      string    `gorm:"unique;not null"`
	ExternalID string    `gorm:"unique"`
}

type RoleBinding struct {
	UserID        uuid.UUID `gorm:"uniqueIndex:userid_workspace"`
	WorkspaceName string    `gorm:"uniqueIndex:userid_workspace"`
	ExternalID    string
	Role          api.Role
	AssignedAt    time.Time
}
