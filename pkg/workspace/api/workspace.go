package api

import (
	"time"

	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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

type ChangeWorkspaceOwnershipRequest struct {
	NewOwnerUserID string `json:"newOwnerUserID"`
}

type ChangeWorkspaceNameRequest struct {
	NewName string `json:"newName"`
}

type WorkspaceResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	OwnerId     uuid.UUID `json:"ownerId"`
	Tier        string    `json:"tier"`
	URI         string    `json:"uri"`
	Status      string    `json:"status"`
	Version     string    `json:"version"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
}

type WorkspaceLimits struct {
	MaxUsers       int64 `json:"maxUsers"`
	MaxConnections int64 `json:"maxConnections"`
	MaxResources   int64 `json:"maxResources"`
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
