package model

import (
	"database/sql"
	"time"
)

type Organization struct {
	OrganizationId string `gorm:"primaryKey"`
	PremiumUntil   *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt sql.NullTime `gorm:"index"`
}
