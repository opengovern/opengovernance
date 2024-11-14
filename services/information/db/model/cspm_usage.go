package model

import (
	"gorm.io/gorm"
	"time"
)

type CspmUsage struct {
	gorm.Model

	GatherTimestamp time.Time `json:"gather_timestamp" gorm:"index:,sort:desc"`

	Hostname             string         `json:"hostname" gorm:"index:ws_id_hostname"`
	IntegrationTypeCount map[string]int `json:"integration_type_count"`
	ApproximateSpend     int            `json:"approximate_spend"`
}
