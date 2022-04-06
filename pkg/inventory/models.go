package inventory

import (
	"gorm.io/gorm"
)

type SmartQuery struct {
	gorm.Model
	Provider    string
	Title       string
	Description string
	Query       string
	Tags        []Tag
}

type Tag struct {
	gorm.Model
	SmartQueryID uint
	Value        string
}
