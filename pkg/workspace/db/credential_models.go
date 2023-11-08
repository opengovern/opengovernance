package db

import (
	"database/sql"
	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"gorm.io/datatypes"
	"time"
)

type Credential struct {
	ID            uuid.UUID      `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()" json:"id"`
	Workspace     string         `gorm:"not null" json:"workspace"`
	ConnectorType source.Type    `gorm:"not null" json:"connectorType"`
	Metadata      datatypes.JSON `json:"metadata,omitempty" gorm:"default:'{}'"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt sql.NullTime `gorm:"index"`
}
