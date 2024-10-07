package models

type QueryView struct {
	ID    string `json:"id" gorm:"primary_key"`
	Query string `json:"query" gorm:"type:text;not null"`
}
