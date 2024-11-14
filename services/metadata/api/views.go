package api

import "time"

type GetViewsCheckpointResponse struct {
	Checkpoint time.Time `json:"checkpoint"`
}
