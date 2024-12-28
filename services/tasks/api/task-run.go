package api

import (
	"github.com/opengovern/opencomply/services/tasks/db/models"
	"time"
)

type TaskRun struct {
	ID             uint      `json:"id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	TaskID         string    `json:"task_id"`
	Status         string    `json:"status"`
	Result         string    `json:"result"`
	FailureMessage string    `json:"failure_message"`
}

type ListTaskRunsResponse struct {
	TotalCount int              `json:"total_count"`
	Items      []models.TaskRun `json:"items"`
}
