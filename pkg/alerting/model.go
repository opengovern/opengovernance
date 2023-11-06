package alerting

import (
	"gorm.io/datatypes"
	"time"
)

type Rule struct {
	Id            uint `json:"id" sql:"AUTO_INCREMENT" gorm:"primary_key"`
	EventType     datatypes.JSON
	Scope         datatypes.JSON
	Operator      datatypes.JSON
	Metadata      datatypes.JSON
	TriggerStatus string
	ActionID      uint `gorm:"foreignKey:action_id"`
}

type Action struct {
	Id      uint `json:"id" sql:"AUTO_INCREMENT" gorm:"primary_key"`
	Name    string
	Method  string
	Url     string
	Headers datatypes.JSON
	Body    string
}

type Triggers struct {
	RuleID         uint
	TriggeredAt    time.Time
	Value          int64
	ResponseStatus int
}
