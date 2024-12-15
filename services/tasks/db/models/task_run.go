package models

import (
	"github.com/jackc/pgtype"
	"github.com/opengovern/opencomply/services/tasks/api"
	"gorm.io/gorm"
)

type TaskRunStatus string

const (
	TaskRunStatusCreated    TaskRunStatus = "CREATED"
	TaskRunStatusQueued     TaskRunStatus = "QUEUED"
	TaskRunStatusInProgress TaskRunStatus = "IN_PROGRESS"
	TaskRunStatusFinished   TaskRunStatus = "FINISHED"
	TaskRunStatusFailed     TaskRunStatus = "FAILED"
	TaskRunStatusTimeout    TaskRunStatus = "TIMEOUT"
)

type TaskRun struct {
	gorm.Model
	TaskID         string
	Params         pgtype.JSONB
	Status         TaskRunStatus
	Result         string
	FailureMessage string
}

func (tr TaskRun) ToAPI() api.TaskRun {
	return api.TaskRun{
		ID:             tr.ID,
		CreatedAt:      tr.CreatedAt,
		UpdatedAt:      tr.UpdatedAt,
		TaskID:         tr.TaskID,
		Status:         string(tr.Status),
		Result:         tr.Result,
		FailureMessage: tr.FailureMessage,
	}
}
