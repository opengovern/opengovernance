package db

import (
	"github.com/jackc/pgtype"
	"github.com/kaytu-io/kaytu-util/pkg/api"
	"gorm.io/gorm"
	"time"
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

type User struct {
	gorm.Model
	Email         string
	EmailVerified bool
	FamilyName    string
	GivenName     string
	Locale        string
	Name          string
	Nickname      string
	Picture       string
	UserId        string
	UserMetadata  pgtype.JSONB
	LastLogin     time.Time
	LastIp        string
	LoginsCount   int
	AppMetadata   pgtype.JSONB
	Username      string
	PhoneNumber   string
	PhoneVerified bool
	Multifactor   []string
	Blocked       bool
}

type WorkspaceMap struct {
	ID   string `gorm:"primaryKey"`
	Name string `gorm:"index"`
}
