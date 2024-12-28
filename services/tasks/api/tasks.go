package api

import "github.com/opengovern/opencomply/services/tasks/db/models"

type TaskListResponse struct {
	Items      []models.Task `json:"items"`
	TotalCount int           `json:"total_count"`
}

type RunTaskRequest struct {
	TaskID string         `json:"task_id"`
	Params map[string]any `json:"params"`
}
