package db

import (
	"time"

	"github.com/jackc/pgtype"
	"gorm.io/gorm"
)

type TaskStatus string

const (
	TaskStatusInProgress TaskStatus = "IN_PROGRESS"
	TaskStatusFinished   TaskStatus = "FINISHED"
	TaskStatusFailed     TaskStatus = "FAILED"
)

type Task struct {
	gorm.Model
	Name              string
	Description       string
	IsCompleted       bool
	LastCompletedDate time.Time
	LastRunDate       time.Time
	ImageUrl          string
	Interval          int
	AutoRun           bool
	Status            TaskStatus
}

type TaskResult struct {
	gorm.Model
	TaskID  uint
	RunDate time.Time
	Result  pgtype.JSONB
}
