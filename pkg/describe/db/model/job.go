package model

import (
	"time"
)

type Job struct {
	ID           uint `gorm:"primarykey"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	JobType      string
	ConnectionID string
	Title        string
	Status       string
}
