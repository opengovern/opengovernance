package model

import (
	"github.com/kaytu-io/kaytu-engine/services/subscription/api/entities"
	"time"
)

type Meter struct {
	WorkspaceID string             `gorm:"primarykey" json:"workspaceId"`
	UsageDate   time.Time          `gorm:"primarykey" json:"usageDate"`
	MeterType   entities.MeterType `gorm:"primarykey" json:"meterType"`

	CreatedAt time.Time `json:"createdAt"`
	Value     int64     `json:"value"`
	Published bool      `json:"-"`
}
