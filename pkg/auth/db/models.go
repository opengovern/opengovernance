package db

import (
	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"gorm.io/gorm"
)

type ApiKey struct {
	gorm.Model
	Name          string
	Role          api.Role
	CreatorUserID string
	WorkspaceID   string
	Active        bool
	Revoked       bool
	MaskedKey     string
	KeyHash       string
}
