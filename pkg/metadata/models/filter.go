package models

type Filters struct {
	Name     string            `json:"name" gorm:"primary_key"`
	KeyValue map[string]string `json:"kay-value" gorm:"key_values"`
}
