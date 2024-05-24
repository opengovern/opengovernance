package model

import (
	"time"
)

type User struct {
	UserId       string `gorm:"primaryKey"`
	PremiumUntil *time.Time
}
