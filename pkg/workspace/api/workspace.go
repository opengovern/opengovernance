package api

import (
	"time"

	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

type BackupStatus struct {
	Phase               v1.BackupPhase     `json:"phase"`
	Progress            *v1.BackupProgress `json:"progress"`
	Expiration          *metav1.Time       `json:"expiration"`
	StartTimestamp      *metav1.Time       `json:"startTimestamp"`
	CompletionTimestamp *metav1.Time       `json:"completionTimestamp"`
	TotalAttempted      int                `json:"totalAttempted"`
	TotalCompleted      int                `json:"totalCompleted"`
	Warnings            int                `json:"warnings"`
	Errors              int                `json:"errors"`
	ValidationErrors    []string           `json:"validationErrors"`
	FailureReason       string             `json:"failureReason"`
}

type RestoreStatus struct {
	Phase               v1.BackupPhase     `json:"phase"`
	Progress            *v1.BackupProgress `json:"progress"`
	Expiration          *metav1.Time       `json:"expiration"`
	StartTimestamp      *metav1.Time       `json:"startTimestamp"`
	CompletionTimestamp *metav1.Time       `json:"completionTimestamp"`
	TotalAttempted      int                `json:"totalAttempted"`
	TotalCompleted      int                `json:"totalCompleted"`
	Warnings            int                `json:"warnings"`
	Errors              int                `json:"errors"`
	ValidationErrors    []string           `json:"validationErrors"`
	FailureReason       string             `json:"failureReason"`
}
