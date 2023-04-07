package db

import (
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
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
