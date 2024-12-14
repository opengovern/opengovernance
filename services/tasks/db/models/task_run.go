package models

import (
	"github.com/jackc/pgtype"
	"gorm.io/gorm"
)

type TaskRunStatus string

const (
	TaskRunStatusCreated    TaskRunStatus = "CREATED"
	TaskRunStatusQueued     TaskRunStatus = "QUEUED"
	TaskRunStatusInProgress TaskRunStatus = "IN_PROGRESS"
	TaskRunStatusFinished   TaskRunStatus = "FINISHED"
	TaskRunStatusFailed     TaskRunStatus = "FAILED"
)

type TaskRun struct {
	gorm.Model
	TaskID         string
	Params         pgtype.JSONB
	Status         TaskRunStatus
	Result         string
	FailureMessage string
}
