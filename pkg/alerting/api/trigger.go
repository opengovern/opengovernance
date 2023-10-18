package api

import (
	"time"
)

type Triggers struct {
	EventType      EventType `json:"event_type"`
	TriggeredAt    time.Time `json:"triggered_at"`
	Scope          Scope     `json:"scope"`
	Value          int64     `json:"value"`
	ResponseStatus int       `json:"response_status"`
}
