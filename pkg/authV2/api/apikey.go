package api

import (
	"github.com/opengovern/og-util/pkg/api"
	"time"
)

type CreateAPIKeyRequest struct {
	Name     string   `json:"name"`                                                 // Name of the key
	Role api.Role `json:"role" enums:"admin,editor,viewer" example:"admin"` // Name of the role
}

type CreateAPIKeyResponse struct {
	ID        uint      `json:"id" example:"1"`                                       // Unique identifier for the key
	Name      string    `json:"name" example:"example"`                               // Name of the key
	Active    bool      `json:"active" example:"true"`                                // Activity state of the key
	CreatedAt time.Time `json:"createdAt" example:"2023-03-31T09:36:09.855Z"`         // Creation timestamp in UTC
	RoleName  api.Role  `json:"roleName" enums:"admin,editor,viewer" example:"admin"` // Name of the role
	Token     string    `json:"token"`                                                // Token of the key
}

type WorkspaceApiKey struct {
	ID            uint      `json:"id" example:"1"`                                       // Unique identifier for the key
	CreatedAt     time.Time `json:"createdAt" example:"2023-03-31T09:36:09.855Z"`         // Creation timestamp in UTC
	UpdatedAt     time.Time `json:"updatedAt" example:"2023-04-21T08:53:09.928Z"`         // Last update timestamp in UTC
	Name          string    `json:"name" example:"example"`                               // Name of the key
	RoleName      api.Role  `json:"roleName" enums:"admin,editor,viewer" example:"admin"` // Name of the role
	CreatorUserID string    `json:"creatorUserID" example:"auth|123456789"`               // Unique identifier of the user who created the key
	Active        bool      `json:"active" example:"true"`                                // Activity state of the key
	MaskedKey     string    `json:"maskedKey" example:"abc...de"`                         // Masked key
}

type UpdateKeyRoleRequest struct {
	ID       uint     `json:"id"`                                                   // Unique identifier for the key
	RoleName api.Role `json:"roleName" enums:"admin,editor,viewer" example:"admin"` // Name of the role
}

type CreateUserRequest struct {
	EmailAddress string    `json:"email_address"`
	Role         *api.Role `json:"role" enums:"admin,editor,viewer" example:"admin"`
	Password     *string   `json:"password"`
	IsActive     bool       `json:"is_active"`

}

type UpdateUserRequest struct {
	EmailAddress string    `json:"email_address"`
	Role         *api.Role `json:"role" enums:"admin,editor,viewer" example:"admin"`
	Password     *string   `json:"password"`
	IsActive     bool       `json:"is_active"`

}


type ResetUserPasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}
