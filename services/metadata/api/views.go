package api

import "time"

type GetViewsCheckpointResponse struct {
	Checkpoint time.Time `json:"checkpoint"`
}

type View struct {
	ID           string   `json:"id"`
	Query        string   `json:"query"`
	Dependencies []string `json:"dependencies"`
}

type GetViewsResponse struct {
	Views []View `json:"views"`
}
