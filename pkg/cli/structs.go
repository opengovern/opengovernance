package cli

import (
	"gitlab.com/keibiengine/keibi-engine/pkg/auth/api"
)

type DeviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationUrl         string `json:"verification_uri"`
	VerificationUrlComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type DeviceCodeRequest struct {
	ClientId string `json:"client_id"`
	Scope    string `json:"scope"`
	Audience string `json:"audience"`
}
type ResponseAccessToken struct {
	AccessToken string `json:"access_token"`
	Scope       string `json:"scope"`
	IdToken     string `json:"id_token"`
	TokenType   string `json:"token_type"`
	ExpireIn    string `json:"expire_in"`
}
type RequestAccessToken struct {
	GrantType  string `json:"grant_type"`
	DeviceCode string `json:"device_code"`
	ClientId   string `json:"client_id"`
}
type Config struct {
	AccessToken string `json:"accessToken"`
}
type ResponseAbout struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"emailVerified"`
}
type RequestGetIamUsers struct {
	Email         string   `json:"email,omitempty"`
	EmailVerified bool     `json:"emailVerified,omitempty"`
	Role          api.Role `json:"role,omitempty"`
}
type RequestCreateUser struct {
	Email string   `json:"email" validate:"required,email"`
	Role  api.Role `json:"role"`
}

type ResponseUserDetails struct {
	UserID        string `json:"userId,omitempty"`        // Unique identifier for the user
	UserName      string `json:"userName,omitempty"`      // Username
	Email         string `json:"email,omitempty"`         // Email address of the user
	EmailVerified bool   `json:"emailVerified,omitempty"` // Is email verified or not
	Role          string `json:"role,omitempty"`          // Name of the role in the specified workspace
	Status        string `json:"status,omitempty"`        // Invite status
	LastActivity  string `json:"lastActivity,omitempty"`  // Last activity timestamp in UTC
	CreatedAt     string `json:"createdAt,omitempty"`     // Creation timestamp in UTC
	Blocked       bool   `json:"blocked,omitempty"`
}
type RolesListResponse struct {
	Role        api.Role `json:"roleName"`
	Description string   `json:"description"`
	UserCount   int      `json:"userCount"`
}
type CountConnectionsCLIRequest struct {
	ConnectorsNames []string `json:"connectores"`
	State           string   `json:"state"`
	Health          string   `json:"health"`
}
type ResponseCreateAzure struct {
	ID [16]byte `json:"ID"`
}

type ResponseAWSCreate struct {
	ID [16]byte `json:"id"`
}
type requestCreateConnectionCredentials struct {
	Config     string `json:"config"`
	Name       string `json:"name"`
	SourceType string `json:"source_Type"`
}
