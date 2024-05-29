package model

import (
	"database/sql"
	"time"
)

type User struct {
	UserId       string `gorm:"primaryKey"`
	PremiumUntil *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt sql.NullTime `gorm:"index"`
}
