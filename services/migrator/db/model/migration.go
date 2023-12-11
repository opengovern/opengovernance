package model

import (
	"time"
)

type Migration struct {
	ID        string `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time

	AdditionalInfo string
}
