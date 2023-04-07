package api

import (
	"time"
)

type CreateAPIKeyRequest struct {
	Name string `json:"name"`
	Role Role   `json:"role"`
}

type CreateAPIKeyResponse struct {
	ID    uint   `json:"id"`
	Token string `json:"token"`
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
