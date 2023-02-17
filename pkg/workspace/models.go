package workspace

import (
	"gitlab.com/keibiengine/keibi-engine/pkg/workspace/api"
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
	StatusSuspending         WorkspaceStatus = "SUSPENDING"
	StatusSuspended          WorkspaceStatus = "SUSPENDED"
)

type Workspace struct {
	gorm.Model

	ID             string   `json:"id"`
	Name           string   `gorm:"uniqueIndex" json:"name"`
	OwnerId        string   `json:"owner_id"`
	URI            string   `json:"uri"`
	Status         string   `json:"status"`
	Description    string   `json:"description"`
	Tier           api.Tier `json:"tier"`
	OrganizationID *int     `json:"organization_id"`
}
