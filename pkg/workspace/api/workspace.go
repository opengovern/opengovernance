package api

import "time"

type CreateWorkspaceRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type CreateWorkspaceResponse struct {
	ID string `json:"id"`
}

type WorkspaceResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	OwnerId     string    `json:"ownerId"`
	Domain      string    `json:"domain"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
}
