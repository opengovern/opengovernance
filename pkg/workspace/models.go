package workspace

import (
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"gorm.io/gorm"
)

type Workspace struct {
	gorm.Model

	ID             string              `json:"id"`
	Name           string              `gorm:"uniqueIndex" json:"name"`
	OwnerId        string              `json:"owner_id"`
	URI            string              `json:"uri"`
	Status         api.WorkspaceStatus `json:"status"`
	Description    string              `json:"description"`
	Tier           api.Tier            `json:"tier"`
	OrganizationID *int                `json:"organization_id"`
}
