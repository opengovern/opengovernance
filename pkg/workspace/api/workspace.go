package api

import (
	"time"

	"github.com/google/uuid"
)

type CreateWorkspaceRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Tier        string `json:"tier"`
}

type CreateWorkspaceResponse struct {
	ID string `json:"id"`
}

type WorkspaceResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	OwnerId     uuid.UUID `json:"ownerId"`
	URI         string    `json:"uri"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
}
