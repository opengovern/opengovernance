package api

import (
	"github.com/opengovern/og-util/pkg/api"
	"time"
)





type InviteStatus string

const (
	InviteStatus_ACCEPTED InviteStatus = "accepted"
	InviteStatus_PENDING  InviteStatus = "pending"
)


type GetUserResponse struct {
	ID        uint       `json:"id" example:"1"`                      // Unique identifier for the user
	UserName      string       `json:"username" example:"John Doe"`                          // Username
	Email         string       `json:"email" example:"johndoe@example.com"`                  // Email address of the user
	EmailVerified bool         `json:"email_verified" example:"true"`                         // Is email verified or not
	RoleName      api.Role     `json:"role_name" enums:"admin,editor,viewer" example:"admin"` // Name of the role
	Status        InviteStatus `json:"status" enums:"accepted,pending" example:"accepted"`   // Invite status
	LastActivity  any   `json:"last_activity" example:"2023-04-21T08:53:09.928Z"`      // Last activity timestamp in UTC
	CreatedAt     time.Time    `json:"createdAt" example:"2023-03-31T09:36:09.855Z"`         // Creation timestamp in UTC
	Blocked       bool         `json:"blocked" example:"false"`                              // Is the user blocked or not
}
type GetUsersResponse struct {
	ID        uint   `json:"id" example:"1"`                      // Unique identifier for the user
	UserName      string   `json:"username" example:"John Doe"`                          // Username
	Email         string   `json:"email" example:"johndoe@example.com"`                  // Email address of the user
	EmailVerified bool     `json:"email_verified" example:"true"`                         // Is email verified or not
	ExternalId	string   `json:"external_id"`
	LastActivity  any   `json:"last_activity" example:"2023-04-21T08:53:09.928Z"`      // Last activity timestamp in UTC
	RoleName      api.Role `json:"role_name" enums:"admin,editor,viewer" example:"admin"` // Name of the role
}

type GetUsersRequest struct {
	Email         *string   `json:"email" example:"johndoe@example.com"`
	EmailVerified *bool     `json:"emailVerified" example:"true"`                         // Filter by
	RoleName      *api.Role `json:"roleName" enums:"admin,editor,viewer" example:"admin"` // Filter by role name
}





type GetMeResponse struct {
	ID          uint              `json:"id" example:"123456789"`                    // Unique identifier for the user
	UserName        string              `json:"username" example:"John Doe"`                        // Username
	Email           string              `json:"email" example:"johndoe@example.com"`                // Email address of the user
	EmailVerified   bool                `json:"email_verified" example:"true"`                       // Is email verified or not
	Status          InviteStatus        `json:"status" enums:"accepted,pending" example:"accepted"` // Invite status
	LastActivity    any         `json:"last_activity" example:"2023-04-21T08:53:09.928Z"`    // Last activity timestamp in UTC
	CreatedAt       time.Time           `json:"created_at" example:"2023-03-31T09:36:09.855Z"`       // Creation timestamp in UTC
	Blocked         bool                `json:"blocked" example:"false"`                            // Is the user blocked or not
	ColorBlindMode  *bool               `json:"color_blind_mode"`
	Role string `json:"role"`
	MemberSince         time.Time         `json:"memberSince"`
	LastLogin       any         `json:"lastLogin"`
}

type UpdateUserRequest struct {
	EmailAddress string    `json:"email_address"`
	Role         *api.Role `json:"role" enums:"admin,editor,viewer" example:"admin"`
	Password     *string   `json:"password"`
	IsActive     bool       `json:"is_active"`
	UserName 	string `json:"username"`
	FullName 	string `json:"full_name"`
	

}

type CreateUserRequest struct {
	EmailAddress string    `json:"email_address"`
	Role         *api.Role `json:"role" enums:"admin,editor,viewer" example:"admin"`
	Password     *string   `json:"password"`
	IsActive     bool       `json:"is_active"`

}

type ResetUserPasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}