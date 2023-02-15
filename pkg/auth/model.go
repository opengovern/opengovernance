package auth

import (
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
)

type RoleBinding struct {
	UserID        string
	WorkspaceName string
	Role          api.Role
	AssignedAt    time.Time
}
