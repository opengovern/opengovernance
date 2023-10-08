package alerting

import (
	"gorm.io/datatypes"
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
