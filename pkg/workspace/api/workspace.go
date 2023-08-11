package api

import "time"

type CreateWorkspaceRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Tier        string `json:"tier"`
}

type CreateWorkspaceResponse struct {
	ID string `json:"id"`
}

type ChangeWorkspaceOwnershipRequest struct {
	NewOwnerUserID string `json:"newOwnerUserID"`
}

type ChangeWorkspaceNameRequest struct {
	NewName string `json:"newName"`
}

type ChangeWorkspaceTierRequest struct {
	NewTier Tier `json:"newName"`
}

type ChangeWorkspaceOrganizationRequest struct {
	NewOrgID uint `json:"newOrgID"`
}

type Workspace struct {
	ID           string          `json:"id" example:"ws-698542025141040315"`
	Name         string          `json:"name" example:"kaytu"`
	OwnerId      string          `json:"ownerId" example:"google-oauth2|204590896945502695694"`
	URI          string          `json:"uri" example:"https://app.kaytu.dev/kaytu"`
	Status       WorkspaceStatus `json:"status" example:"PROVISIONED"`
	Description  string          `json:"description" example:"Lorem ipsum dolor sit amet, consectetur adipiscing elit."`
	Tier         Tier            `json:"tier" example:"ENTERPRISE"`
	Organization Organization    `json:"organization,omitempty"`
	CreatedAt    time.Time       `json:"createdAt" example:"2023-05-17T14:39:02.707659Z"`
}

type WorkspaceResponse struct {
	Workspace
	Version string `json:"version" example:"v0.45.4"`
}

type Organization struct {
	ID           uint   `json:"id"`
	CompanyName  string `json:"companyName"`
	Url          string `json:"url"`
	Address      string `json:"address"`
	City         string `json:"city"`
	State        string `json:"state"`
	Country      string `json:"country"`
	ContactPhone string `json:"contactPhone"`
	ContactEmail string `json:"contactEmail"`
	ContactName  string `json:"contactName"`
}

type WorkspaceLimits struct {
	MaxUsers       int64 `json:"maxUsers"`
	MaxConnections int64 `json:"maxConnections"`
	MaxResources   int64 `json:"maxResources"`
}

type WorkspaceLimitsUsage struct {
	ID   string `json:"id" example:"ws-698542025141040315"`
	Name string `json:"name" example:"kaytu"`

	CurrentUsers       int64 `json:"currentUsers" example:"10"`
	CurrentConnections int64 `json:"currentConnections" example:"100"`
	CurrentResources   int64 `json:"currentResources" example:"10000"`

	MaxUsers       int64 `json:"maxUsers" example:"100"`
	MaxConnections int64 `json:"maxConnections" example:"1000"`
	MaxResources   int64 `json:"maxResources" example:"100000"`
}
