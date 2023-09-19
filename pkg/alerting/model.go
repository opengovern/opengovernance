package alerting

import "encoding/json"

type Rule struct {
	ID        uint `gorm:"primaryKey"`
	EventType json.RawMessage
	Scope     json.RawMessage
	Operator  string
	Value     int64
	ActionID  uint `gorm:"foreignkey:ActionID"`
}

type Action struct {
	ID      uint `gorm:"primaryKey"`
	Method  string
	Url     string
	Headers json.RawMessage
	Body    string
}
