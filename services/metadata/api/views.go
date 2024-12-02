package api

import (
	"time"
)

type GetViewsCheckpointResponse struct {
	Checkpoint time.Time `json:"checkpoint"`
}

type View struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Query        Query    `json:"query"`
	Dependencies []string `json:"dependencies"`
}

type GetViewsResponse struct {
	Views []View `json:"views"`
}

type Query struct {
	ID             string       `json:"id"`
	QueryToExecute string       `json:"query_to_execute"`
	PrimaryTable   *string      `json:"primary_table"`
	ListOfTables   []string     `json:"list_of_tables"`
	Engine         string       `json:"engine"`
	Parameters     []Parameters `json:"parameters"`
	Global         bool         `json:"global"`
}

type Parameters struct {
	Key      string `gorm:"primaryKey"`
	Required bool   `gorm:"not null"`
}
