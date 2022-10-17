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
	Tags        []Tag `gorm:"many2many:smartquery_tags;"`
}

type Tag struct {
	gorm.Model
	Key          string
	Value        string
	SmartQueries []SmartQuery `gorm:"many2many:smartquery_tags;"`
}

type Category struct {
	gorm.Model
	Name         string
	SubCategory  string
	Cloud        string
	CloudService string
}
