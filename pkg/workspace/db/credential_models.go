package db

import (
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Credential struct {
	gorm.Model

	WorkspaceID     string         `gorm:"not null" json:"workspaceID"`
	ConnectorType   source.Type    `gorm:"not null" json:"connectorType"`
	Metadata        datatypes.JSON `json:"metadata,omitempty" gorm:"default:'{}'"`
	IsCreated       bool           `gorm:"default:false" json:"is_created"`
	ConnectionCount int
}

type MasterCredential struct {
	gorm.Model

	WorkspaceID   string      `gorm:"not null" json:"workspaceID"`
	ConnectorType source.Type `gorm:"not null" json:"connectorType"`
	Credential    string
}
