package model

import "time"

type DataAge struct {
	DataType  string `gorm:"primaryKey"`
	UpdatedAt time.Time
}
