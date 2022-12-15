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
	Access map[string]api.Role `json:"access"`
}

type User struct {
	CreatedAt     time.Time `json:"created_at"`
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
	Name         string    `json:"name"`
	Nickname     string    `json:"nickname"`
	Picture      string    `json:"picture"`
	UpdatedAt    time.Time `json:"updated_at"`
	UserId       string    `json:"user_id"`
	UserMetadata Metadata  `json:"user_metadata"`
	LastLogin    time.Time `json:"last_login"`
	LastIp       string    `json:"last_ip"`
	LoginsCount  int       `json:"logins_count"`
	AppMetadata  Metadata  `json:"app_metadata"`
}
