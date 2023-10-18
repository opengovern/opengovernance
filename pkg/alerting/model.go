package alerting

import (
	"gorm.io/datatypes"
	"time"
)

type Rule struct {
	Id        uint `json:"id" sql:"AUTO_INCREMENT" gorm:"primary_key"`
	EventType datatypes.JSON
	Scope     datatypes.JSON
	Operator  datatypes.JSON
	ActionID  uint `gorm:"foreignKey:action_id"`
}

type Action struct {
	Id      uint `json:"id" sql:"AUTO_INCREMENT" gorm:"primary_key"`
	Method  string
	Url     string
	Headers datatypes.JSON
	Body    string
}

type TriggerCompliance struct {
	ComplianceId   string `json:"compliance_id" gorm:"primary_key"`
	Hour           time.Time
	ConnectionId   string
	Value          int64
	ResponseStatus int
}

type TriggerInsight struct {
	InsightId      int64 `json:"insight_id" gorm:"primary_key"`
	Hour           time.Time
	ConnectionId   string
	Value          int64
	ResponseStatus int
}
