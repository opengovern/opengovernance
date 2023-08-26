package models

type Filter struct {
	Name     string            `json:"name" gorm:"primary_key"`
	KeyValue map[string]string `json:"kayValue" gorm:"key_values"`
}
