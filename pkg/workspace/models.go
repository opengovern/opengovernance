package workspace

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WorkspaceStatus string

func (ws WorkspaceStatus) String() string {
	return string(ws)
}

const (
	StatusProvisioning       WorkspaceStatus = "PROVISIONING"
	StatusProvisioned        WorkspaceStatus = "PROVISIONED"
	StatusProvisioningFailed WorkspaceStatus = "PROVISIONING_FAILED"
	StatusDeleting           WorkspaceStatus = "DELETING"
	StatusDeleted            WorkspaceStatus = "DELETED"
)

type Workspace struct {
	gorm.Model

	ID          uuid.UUID `json:"id"`
	Name        string    `gorm:"uniqueIndex" json:"name"`
	OwnerId     uuid.UUID `json:"owner_id"`
	Domain      string    `json:"domain"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
}
