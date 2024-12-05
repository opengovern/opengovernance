package models

import (
	"github.com/lib/pq"
	"github.com/opengovern/og-util/pkg/model"
	"time"
)

type QueryViewTag struct {
	model.Tag
	QueryViewID string `gorm:"primaryKey"`
}

type QueryView struct {
	ID           string `json:"id" gorm:"primary_key"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	QueryID      *string
	Query        *Query         `gorm:"foreignKey:QueryID;references:ID;constraint:OnDelete:SET NULL"`
	Dependencies pq.StringArray `gorm:"type:text[]"`
	Tags         []QueryViewTag `gorm:"foreignKey:QueryViewID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

type Query struct {
	ID              string `gorm:"primaryKey"`
	QueryToExecute  string
	IntegrationType pq.StringArray `gorm:"type:text[]"`
	PrimaryTable    *string
	ListOfTables    pq.StringArray `gorm:"type:text[]"`
	Engine          string
	QueryViews      []QueryView      `gorm:"foreignKey:QueryID"`
	Parameters      []QueryParameter `gorm:"foreignKey:QueryID"`
	Global          bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type QueryParameter struct {
	QueryID  string `gorm:"primaryKey"`
	Key      string `gorm:"primaryKey"`
	Required bool   `gorm:"default:false"`
}
