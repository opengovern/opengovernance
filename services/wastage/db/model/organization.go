package model

import (
	"time"
)

type Organization struct {
	OrganizationId string `gorm:"primaryKey"`
	PremiumUntil   *time.Time
}
