package models

import (
	"github.com/jackc/pgtype"
	"time"

	"gorm.io/gorm"
)

type Task struct {
	gorm.Model
	Name              string `gorm:"unique;not null"` // Enforces uniqueness and non-null constraint
	Description       string
	LastCompletedDate time.Time
	LastRunDate       time.Time
	ImageUrl          string
	Interval          uint64
	NatsConfig        pgtype.JSONB
	ScaleConfig       pgtype.JSONB
}
