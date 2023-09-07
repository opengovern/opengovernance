package api

import "encoding/json"

type ResponseRule struct {
	ID        uint            `json:"id"`
	EventType json.RawMessage `json:"event_type"`
	Scope     json.RawMessage `json:"scope"`
	Operator  string          `json:"operator"`
	Value     int64           `json:"value"`
	ActionID  uint            `json:"action_id"`
}

type RequestRule struct {
	ID        uint            `json:"id"`
	EventType json.RawMessage `json:"event_type"`
	Scope     json.RawMessage `json:"scope"`
	Operator  string          `json:"operator"`
	Value     int64           `json:"value"`
	ActionID  uint            `json:"action_id"`
}
