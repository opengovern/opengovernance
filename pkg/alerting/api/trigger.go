package api

import (
	"time"
)

type Triggers struct {
	Rule           Rule
	Action         Action
	TriggeredAt    time.Time `json:"triggered_at"`
	Value          int64     `json:"value"`
	ResponseStatus int       `json:"response_status"`
}
