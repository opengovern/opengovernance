package db

import (
	"github.com/google/uuid"
	"github.com/jackc/pgtype"
	"github.com/lib/pq"
	"github.com/opengovern/og-util/pkg/api"
	"gorm.io/gorm"
	"time"
)

type UserLifecycle string

const (
	UserLifecycleActive   UserLifecycle = "active"
	UserLifecycleInactive UserLifecycle = "inactive"
	UserLifecycleBlocked  UserLifecycle = "blocked"
	UserLifecycleDeleted  UserLifecycle = "deleted"
	UserLifecycleNone     UserLifecycle = ""
)

type Configuration struct {
	gorm.Model
	Key   string
	Value string
}

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
	UserUuid              uuid.UUID
	Email                 string
	EmailVerified         bool
	StaticOwner           bool
	FamilyName            string
	GivenName             string
	Locale                string
	Name                  string
	Nickname              string
	Picture               string
	UserId                string
	IdLifecycle           UserLifecycle
	Role                  api.Role
	ConnectorId           string
	ExternalId            string
	UserMetadata          pgtype.JSONB
	LastLogin             time.Time
	LastIp                string
	LoginsCount           int
	AppMetadata           pgtype.JSONB
	Username              string
	PhoneNumber           string
	PhoneVerified         bool
	Multifactor           pq.StringArray `gorm:"type:text[]"`
	Blocked               bool
	RequirePasswordChange bool `gorm:"default:true"`
	Connector             string
	Disabled              bool
}

type WorkspaceMap struct {
	ID   string `gorm:"primaryKey"`
	Name string `gorm:"index"`
}
