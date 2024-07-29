package db

import (
	"encoding/json"
	"github.com/jackc/pgtype"
	"github.com/kaytu-io/kaytu-engine/pkg/auth/auth0"
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

func (u *User) ToApi() (*auth0.User, error) {
	userMetadata := auth0.Metadata{}
	appMetadata := auth0.Metadata{}

	err := json.Unmarshal(u.UserMetadata.Bytes, &userMetadata)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(u.AppMetadata.Bytes, &appMetadata)
	if err != nil {
		return nil, err
	}

	return &auth0.User{
		Email:         u.Email,
		EmailVerified: u.EmailVerified,
		FamilyName:    u.FamilyName,
		GivenName:     u.GivenName,
		Locale:        u.Locale,
		Name:          u.Name,
		Nickname:      u.Nickname,
		Picture:       u.Picture,
		UserId:        u.UserId,
		UserMetadata:  userMetadata,
		LastLogin:     u.LastLogin,
		LastIp:        u.LastIp,
		LoginsCount:   u.LoginsCount,
		AppMetadata:   appMetadata,
		Username:      u.Username,
		PhoneNumber:   u.PhoneNumber,
		PhoneVerified: u.PhoneVerified,
		Multifactor:   u.Multifactor,
		Blocked:       u.Blocked,
	}, nil
}

type WorkspaceMap struct {
	ID   string `gorm:"primaryKey"`
	Name string `gorm:"index"`
}
