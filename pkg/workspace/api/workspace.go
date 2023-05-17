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
	ID           string                `json:"id"`
	Name         string                `json:"name"`
	OwnerId      string                `json:"ownerId"`
	Tier         string                `json:"tier"`
	URI          string                `json:"uri"`
	Status       string                `json:"status"`
	Version      string                `json:"version"`
	Description  string                `json:"description"`
	CreatedAt    time.Time             `json:"createdAt"`
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
	ID             string `json:"id"`
	Name           string `gorm:"uniqueIndex" json:"name"`
	OwnerId        string `json:"owner_id"`
	URI            string `json:"uri"`
	Status         string `json:"status"`
	Description    string `json:"description"`
	Tier           Tier   `json:"tier"`
	OrganizationID *int   `json:"organization_id"`
}

type WorkspaceLimitsUsage struct {
	ID   string `json:"id"`
	Name string `json:"name"`

	CurrentUsers       int64 `json:"currentUsers"`
	CurrentConnections int64 `json:"currentConnections"`
	CurrentResources   int64 `json:"currentResources"`

	MaxUsers       int64 `json:"maxUsers"`
	MaxConnections int64 `json:"maxConnections"`
	MaxResources   int64 `json:"maxResources"`
}
