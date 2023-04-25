package cli

import (
	"time"
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
	Email         string `json:"email"`
	EmailVerified bool   `json:"emailVerified"`
	Role          string `json:"role"`
}
type ResponseGetIamUsers struct {
	Blocked       bool   `json:"blocked"`
	CreateAt      string `json:"createAt"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"emailVerified"`
	LastActivity  string `json:"lastActivity"`
	Role          string `json:"role"`
	Status        string `json:"status"`
	UserId        string `json:"userId"`
	UserName      string `json:"userName"`
}

type ResponseListRoles struct {
	Role        string `json:"role"`
	Description string `json:"description"`
	UserCount   int    `json:"user-count"`
}
type RoleDetailsResponse struct {
	Role        string
	Description string
	UserCount   int
	Users       []GetUserResponse
}
type GetUserResponse struct {
	UserID        string    `json:"userId"`        // Unique identifier for the user
	UserName      string    `json:"userName"`      // Username
	Email         string    `json:"email"`         // Email address of the user
	EmailVerified bool      `json:"emailVerified"` // Is email verified or not
	Role          string    `json:"role"`          // Name of the role in the specified workspace
	Status        string    `json:"status"`        // Invite status
	LastActivity  time.Time `json:"lastActivity"`  // Last activity timestamp in UTC
	CreatedAt     time.Time `json:"createdAt"`     // Creation timestamp in UTC
	Blocked       bool      `json:"blocked"`       // Is the user blocked or not
}
type RequestCreateUser struct {
	Email string `json:"email"`
	Role  string `json:"role"`
}
type RequestCreateKey struct {
	Name string `json:"name"`
	Role string `json:"role"`
}
type RequestUpdateUser struct {
	Role   string `json:"role"`
	UserId string `json:"userId"`
}
type ResponseUserDetails struct {
	UserID        string `json:"userId"`        // Unique identifier for the user
	UserName      string `json:"userName"`      // Username
	Email         string `json:"email"`         // Email address of the user
	EmailVerified bool   `json:"emailVerified"` // Is email verified or not
	Role          string `json:"role"`          // Name of the role in the specified workspace
	Status        string `json:"status"`        // Invite status
	LastActivity  string `json:"lastActivity"`  // Last activity timestamp in UTC
	CreatedAt     string `json:"createdAt"`     // Creation timestamp in UTC
	Blocked       bool
}
