package api

import (
	"time"
)

type CreateAPIKeyRequest struct {
	Name string `json:"name"`
	Role Role   `json:"role"`
}

type CreateAPIKeyResponse struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"createdAt"`
	Role      Role      `json:"role"`
	Token     string    `json:"token"`
}

type WorkspaceApiKey struct {
	ID            uint      `json:"id"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	Name          string    `json:"name"`
	Role          Role      `json:"role"`
	CreatorUserID string    `json:"creatorUserID"`
	Active        bool      `json:"active"`
	MaskedKey     string    `json:"maskedKey"`
}

type UpdateKeyRoleRequest struct {
	ID   uint `json:"id"`
	Role Role `json:"role"`
}
