package alerting

import "encoding/json"

type Rule struct {
	ID        uint
	EventType json.RawMessage
	Scope     json.RawMessage
	Operator  string
	Value     int64
	ActionID  uint
}

type Action struct {
	ID      uint
	Method  string
	Url     string
	Headers json.RawMessage
	Body    string
}
