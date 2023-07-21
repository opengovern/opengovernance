package api

import (
	"time"
)

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
	NewOrgID int `json:"newOrgID"`
}

type WorkspaceResponse struct {
	ID           string                `json:"id" example:"ws-698542025141040315"`
	Name         string                `json:"name" example:"keibi"`
	OwnerId      string                `json:"ownerId" example:"google-oauth2|204590896945502695694"`
	Tier         Tier                  `json:"tier" example:"ENTERPRISE"`
	URI          string                `json:"uri" example:"https://app.kaytu.dev/keibi"`
	Status       WorkspaceStatus       `json:"status" example:"PROVISIONED"`
	Version      string                `json:"version" example:"v0.45.4"`
	Description  string                `json:"description" example:"Lorem ipsum dolor sit amet, consectetur adipiscing elit."`
	CreatedAt    time.Time             `json:"createdAt" example:"2023-05-17T14:39:02.707659Z"`
	Organization *OrganizationResponse `json:"organization,omitempty"`
}

type OrganizationResponse struct {
	ID            int    `json:"id"`
	CompanyName   string `json:"companyName"`
	Url           string `json:"url"`
	AddressLine1  string `json:"addressLine1"`
	AddressLine2  string `json:"addressLine2"`
	AddressLine3  string `json:"addressLine3"`
	City          string `json:"city"`
	State         string `json:"state"`
	Country       string `json:"country"`
	ContactPhone  string `json:"contactPhone"`
	ContactEmail  string `json:"contactEmail"`
	ContactPerson string `json:"contactPerson"`
}

type WorkspaceLimits struct {
	MaxUsers       int64 `json:"maxUsers"`
	MaxConnections int64 `json:"maxConnections"`
	MaxResources   int64 `json:"maxResources"`
}

type Workspace struct {
	ID             string          `json:"id"`
	Name           string          `gorm:"uniqueIndex" json:"name"`
	OwnerId        string          `json:"owner_id"`
	URI            string          `json:"uri"`
	Status         WorkspaceStatus `json:"status"`
	Description    string          `json:"description"`
	Tier           Tier            `json:"tier"`
	OrganizationID *int            `json:"organization_id"`
}

type WorkspaceLimitsUsage struct {
	ID   string `json:"id" example:"ws-698542025141040315"`
	Name string `json:"name" example:"keibi"`

	CurrentUsers       int64 `json:"currentUsers" example:"10"`
	CurrentConnections int64 `json:"currentConnections" example:"100"`
	CurrentResources   int64 `json:"currentResources" example:"10000"`

	MaxUsers       int64 `json:"maxUsers" example:"100"`
	MaxConnections int64 `json:"maxConnections" example:"1000"`
	MaxResources   int64 `json:"maxResources" example:"100000"`
}
