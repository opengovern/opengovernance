package alerting

import (
	"gorm.io/datatypes"
)

type Rule struct {
	ID        uint `gorm:"primaryKey"`
	EventType datatypes.JSON
	Scope     datatypes.JSON
	Operator  datatypes.JSON
	ActionID  uint `gorm:"foreignKey:action_id"`
}

type Action struct {
	ID      uint `gorm:"primaryKey"`
	Method  string
	Url     string
	Headers datatypes.JSON
	Body    string
}
