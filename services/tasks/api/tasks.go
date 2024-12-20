package api

type TaskListResponse struct {
	Tasks      []TaskResponse `json:"tasks"`
	TotalCount int            `json:"total_count"`
}

type TaskResponse struct {
	ID            uint   `json:"id"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	IsCompleted   bool   `json:"is_completed"`
	CompletedDate string `json:"completed_date"`
	LastRunDate   string `json:"last_run_date"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
	ImageUrl      string `json:"image_url"`
	Interval      int    `json:"interval"`
	Status        string `json:"status"`
	Autorun       bool   `json:"autorun"`
}

type RunTaskRequest struct {
	TaskID string              `json:"task_id"`
	Params map[string][]string `json:"params"`
}
