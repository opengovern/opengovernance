package db

import (
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"gorm.io/gorm"
)

type Credential struct {
	gorm.Model

	WorkspaceID      string      `gorm:"not null" json:"workspaceID"`
	ConnectorType    source.Type `gorm:"not null" json:"connectorType"`
	Metadata         string      `json:"metadata,omitempty"`
	IsCreated        bool        `gorm:"default:false" json:"is_created"`
	ConnectionCount  int
	SingleConnection bool
}

type MasterCredential struct {
	gorm.Model

	WorkspaceID   string      `gorm:"not null" json:"workspaceID"`
	ConnectorType source.Type `gorm:"not null" json:"connectorType"`
	Credential    string
}
