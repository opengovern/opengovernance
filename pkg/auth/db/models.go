package db

import (
	"github.com/opengovern/og-util/pkg/api"
	"gorm.io/gorm"
	"time"
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
	IsActive      bool
	KeyHash       string
	MaskedKey     string
}

type Connector struct {
	gorm.Model
	UserNumber 		uint
	IsActive bool   `gorm:"default:true"`
	ConnectorID string
	ConnectorType string
	ConnectorSubType string	
	LastUpdate 		time.Time
	
}

type User struct {
	gorm.Model
	Email                 string
	EmailVerified         bool
	FullName              string
	Role                  api.Role
	ConnectorId           string
	ExternalId            string
	LastLogin             time.Time
	Username              string
	RequirePasswordChange bool `gorm:"default:true"`
	Connector             string
	IsActive              bool `gorm:"default:true"`
}
