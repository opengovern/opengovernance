package api

import "time"

type CreateWorkspaceRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type CreateWorkspaceResponse struct {
	WorkspaceId string `json:"workspaceId"`
}

type WorkspaceResponse struct {
	WorkspaceId string    `json:"workspaceId"`
	Name        string    `json:"name"`
	OwnerId     string    `json:"ownerId"`
	Domain      string    `json:"domain"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
}
