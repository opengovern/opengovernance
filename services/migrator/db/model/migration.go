package model

import (
	"github.com/jackc/pgtype"
	"time"
)

type JobsStatus string

const (
	JobStatusCompleted  JobsStatus = "SUCCEEDED"
	JobStatusPending    JobsStatus = "PENDING"
	JobStatusInProgress JobsStatus = "IN_PROGRESS"
	JobStatusFailed     JobsStatus = "FAILED"
)

type Migration struct {
	ID         string `gorm:"primarykey"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Status     string
	JobsStatus pgtype.JSONB

	AdditionalInfo string
}
