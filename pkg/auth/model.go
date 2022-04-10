package auth

import (
	"time"

	"github.com/lib/pq"
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
)

type RoleBinding struct {
	UserID     string `gorm:"primaryKey"`
	Name       string
	Emails     pq.StringArray `gorm:"type:text[]"`
	Role       api.Role
	AssignedAt time.Time
}
