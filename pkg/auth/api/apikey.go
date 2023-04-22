package api

import (
	"time"
)

type CreateAPIKeyRequest struct {
	Name     string `json:"name"`
	RoleName Role   `json:"roleName" enums:"admin,editor,viewer" example:"admin"` // Name of the role
}

type CreateAPIKeyResponse struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Active    bool      `json:"acrive"`
	CreatedAt time.Time `json:"createdAt"`
	RoleName  Role      `json:"roleName" enums:"admin,editor,viewer" example:"admin"` // Name of the role
	Token     string    `json:"token"`
}

type WorkspaceApiKey struct {
	ID            uint      `json:"id"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	Name          string    `json:"name"`
	RoleName      Role      `json:"roleName" enums:"admin,editor,viewer" example:"admin"` // Name of the role
	CreatorUserID string    `json:"creatorUserID"`
	Active        bool      `json:"active"`
	MaskedKey     string    `json:"maskedKey"`
}

type UpdateKeyRoleRequest struct {
	ID       uint `json:"id"`
	RoleName Role `json:"roleName" enums:"admin,editor,viewer" example:"admin"` // Name of the role
}
