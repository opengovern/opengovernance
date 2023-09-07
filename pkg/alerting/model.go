package alerting

import "encoding/json"

type Rule struct {
	ID        uint            `json:"id"`
	EventType json.RawMessage `json:"event_type"`
	Scope     json.RawMessage `json:"scope"`
	Operator  *string         `json:"operator"`
	Value     *int64          `json:"value"`
	ActionID  *uint           `json:"action_id"`
}

type Action struct {
	ID      uint            `json:"id"`
	Method  *string         `json:"method"`
	Url     *string         `json:"url"`
	Headers json.RawMessage `json:"headers"`
	Body    *string         `json:"body"`
}
