package model

import (
	"database/sql"
	"github.com/opengovern/og-util/pkg/source"
	"gorm.io/datatypes"
	"time"
)

type Tier string

const (
	Tier_Community  Tier = "Community"
	Tier_Enterprise Tier = "Enterprise"
)

type Connector struct {
	ID                  uint
	Name                source.Type                   `gorm:"primaryKey" json:"name"`
	Label               string                        `json:"label"`
	ShortDescription    string                        `json:"short_description"`
	Description         string                        `json:"description"`
	Direction           source.ConnectorDirectionType `gorm:"default:'ingress'" json:"direction"`
	Status              source.ConnectorStatus        `gorm:"default:'enabled'" json:"status"`
	Tier                Tier                          `gorm:"default:'Community'" json:"tier"`
	Logo                string                        `gorm:"default:''" json:"logo"`
	AutoOnboardSupport  bool                          `gorm:"default:false" json:"auto_onboard_support"`
	AllowNewConnections bool                          `gorm:"default:true" json:"allow_new_connections"`
	MaxConnectionLimit  int                           `gorm:"default:25" json:"max_connection_limit"`
	Tags                datatypes.JSON                `gorm:"default:'{}'" json:"tags"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt sql.NullTime `gorm:"index"`
}
