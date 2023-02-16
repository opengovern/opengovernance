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
	WorkspaceAccess map[string]api.Role `json:"workspaceAccess"`
	GlobalAccess    *api.Role           `json:"globalAccess"`
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
	Email         string   `json:"email"`
	PhoneNumber   string   `json:"phone_number"`
	UserMetadata  Metadata `json:"user_metadata"`
	Blocked       bool     `json:"blocked"`
	EmailVerified bool     `json:"email_verified"`
	PhoneVerified bool     `json:"phone_verified"`
	AppMetadata   Metadata `json:"app_metadata"`
	GivenName     string   `json:"given_name"`
	FamilyName    string   `json:"family_name"`
	Name          string   `json:"name"`
	Nickname      string   `json:"nickname"`
	Picture       string   `json:"picture"`
	UserId        string   `json:"user_id"`
	Connection    string   `json:"connection"`
	Password      string   `json:"password"`
	VerifyEmail   bool     `json:"verify_email"`
	Username      string   `json:"username"`
}

type CreatePasswordChangeTicketRequest struct {
	ResultUrl              string `json:"result_url"`
	UserId                 string `json:"user_id"`
	ClientId               string `json:"client_id"`
	OrganizationId         string `json:"organization_id"`
	ConnectionId           string `json:"connection_id"`
	Email                  string `json:"email"`
	TtlSec                 int    `json:"ttl_sec"`
	MarkEmailAsVerified    bool   `json:"mark_email_as_verified"`
	IncludeEmailInRedirect bool   `json:"includeEmailInRedirect"`
}

type CreatePasswordChangeTicketResponse struct {
	Ticket string `json:"ticket"`
}
