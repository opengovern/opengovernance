package model

import (
	"gorm.io/gorm"
)

type Organization struct {
	gorm.Model

	OrganizationId string `gorm:"primaryKey"`
	Premium        bool
}
