package auth0

import (
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
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
	WorkspaceAccess map[string]api.Role `json:"workspaceAccess,omitempty"`
	GlobalAccess    *api.Role           `json:"globalAccess,omitempty"`
}

type User struct {
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Email         string    `json:"email"`
	EmailVerified bool      `json:"email_verified"`
	FamilyName    string    `json:"family_name"`
	GivenName     string    `json:"given_name"`
	Locale        string    `json:"locale"`
	Identities    []struct {
		Connection string `json:"connection"`
		Provider   string `json:"provider"`
		UserId     string `json:"user_id"`
		IsSocial   bool   `json:"isSocial"`
	} `json:"identities"`
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
