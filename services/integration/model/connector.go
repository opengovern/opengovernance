package model

import (
	"database/sql"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"gorm.io/datatypes"
	"time"
)

type Connector struct {
	Name                source.Type `gorm:"primaryKey"`
	Label               string
	ShortDescription    string
	Description         string
	Direction           source.ConnectorDirectionType `gorm:"default:'ingress'"`
	Status              source.ConnectorStatus        `gorm:"default:'enabled'"`
	Logo                string                        `gorm:"default:''"`
	AutoOnboardSupport  bool                          `gorm:"default:false"`
	AllowNewConnections bool                          `gorm:"default:true"`
	MaxConnectionLimit  int                           `gorm:"default:25"`
	Tags                datatypes.JSON                `gorm:"default:'{}'"`

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt sql.NullTime `gorm:"index"`
}
