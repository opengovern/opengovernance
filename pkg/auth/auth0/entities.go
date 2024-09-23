package auth0

import (
	"encoding/json"
	api2 "github.com/kaytu-io/kaytu-util/pkg/api"
	"github.com/kaytu-io/open-governance/pkg/auth/db"
	"time"

	"github.com/kaytu-io/open-governance/pkg/auth/api"
)

type TokenRequest struct {
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Audience     string `json:"audience"`
	GrantType    string `json:"grant_type"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	ExpiresIn   int    `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

type Metadata struct {
	WorkspaceAccess map[string]api2.Role `json:"workspaceAccess"`
	GlobalAccess    *api2.Role           `json:"globalAccess,omitempty"`
	ColorBlindMode  *bool                `json:"colorBlindMode,omitempty"`
	Theme           *api.Theme           `json:"theme,omitempty"`
	MemberSince     *string              `json:"memberSince,omitempty"`
	LastLogin       *string              `json:"userLastLogin,omitempty"`
	ConnectionIDs   map[string][]string  `json:"connectionIDs"`
}

type User struct {
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Email         string    `json:"email"`
	EmailVerified bool      `json:"email_verified"`
	FamilyName    string    `json:"family_name"`
	GivenName     string    `json:"given_name"`
	Locale        string    `json:"locale"`
	Name          string    `json:"name"`
	Nickname      string    `json:"nickname"`
	Picture       string    `json:"picture"`
	UserId        string    `json:"user_id"`
	UserMetadata  Metadata  `json:"user_metadata"`
	LastLogin     time.Time `json:"last_login"`
	LastIp        string    `json:"last_ip"`
	LoginsCount   int       `json:"logins_count"`
	AppMetadata   Metadata  `json:"app_metadata"`
	Username      string    `json:"username"`
	PhoneNumber   string    `json:"phone_number"`
	PhoneVerified bool      `json:"phone_verified"`
	Multifactor   []string  `json:"multifactor"`
	Blocked       bool      `json:"blocked"`
}

func DbUserToApi(u *db.User) (*User, error) {
	if u == nil {
		return nil, nil
	}
	userMetadata := Metadata{}
	appMetadata := Metadata{}

	if len(u.UserMetadata.Bytes) > 0 {
		err := json.Unmarshal(u.UserMetadata.Bytes, &userMetadata)
		if err != nil {
			return nil, err
		}
	}

	if len(u.AppMetadata.Bytes) > 0 {
		err := json.Unmarshal(u.AppMetadata.Bytes, &appMetadata)
		if err != nil {
			return nil, err
		}
	}

	return &User{
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
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

type CreateUserRequest struct {
	Email         string   `json:"email,omitempty"`
	PhoneNumber   string   `json:"phone_number,omitempty"`
	UserMetadata  Metadata `json:"user_metadata,omitempty"`
	Blocked       bool     `json:"blocked,omitempty"`
	EmailVerified bool     `json:"email_verified,omitempty"`
	PhoneVerified bool     `json:"phone_verified,omitempty"`
	AppMetadata   Metadata `json:"app_metadata,omitempty"`
	GivenName     string   `json:"given_name,omitempty"`
	FamilyName    string   `json:"family_name,omitempty"`
	Name          string   `json:"name,omitempty"`
	Nickname      string   `json:"nickname,omitempty"`
	Picture       string   `json:"picture,omitempty"`
	UserId        string   `json:"user_id,omitempty"`
	Connection    string   `json:"connection,omitempty"`
	Password      string   `json:"password,omitempty"`
	VerifyEmail   bool     `json:"verify_email,omitempty"`
	Username      string   `json:"username,omitempty"`
}

type CreatePasswordChangeTicketRequest struct {
	ResultUrl              string `json:"result_url,omitempty"`
	UserId                 string `json:"user_id,omitempty"`
	ClientId               string `json:"client_id,omitempty"`
	OrganizationId         string `json:"organization_id,omitempty"`
	ConnectionId           string `json:"connection_id,omitempty"`
	Email                  string `json:"email,omitempty"`
	TTLSec                 int    `json:"ttl_sec,omitempty"`
	MarkEmailAsVerified    bool   `json:"mark_email_as_verified,omitempty"`
	IncludeEmailInRedirect bool   `json:"includeEmailInRedirect,omitempty"`
}

type CreatePasswordChangeTicketResponse struct {
	Ticket string `json:"ticket"`
}
