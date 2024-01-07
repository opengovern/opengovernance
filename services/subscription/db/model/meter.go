package model

import (
	"github.com/kaytu-io/kaytu-engine/services/subscription/api/entities"
	"time"
)

type Meter struct {
	WorkspaceID string             `gorm:"primarykey"`
	UsageDate   time.Time          `gorm:"primarykey"`
	MeterType   entities.MeterType `gorm:"primarykey"`

	CreatedAt time.Time
	Value     int64
	Published bool
}
