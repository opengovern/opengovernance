package model

type ConnectionGroup struct {
	Name  string `gorm:"primaryKey" json:"name"`
	Query string `json:"query"`
}
