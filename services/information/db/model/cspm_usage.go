package model

import (
	"gorm.io/gorm"
	"time"
)

type CspmUsage struct {
	gorm.Model

	WorkspaceId     string    `json:"workspace_id" gorm:"index:ws_id_hostname"`
	GatherTimestamp time.Time `json:"gather_timestamp" gorm:"index:,sort:desc"`

	Hostname               string `json:"hostname" gorm:"index:ws_id_hostname"`
	AwsAccountCount        int    `json:"aws_account_count"`
	AzureSubscriptionCount int    `json:"azure_account_count"`
	ApproximateSpend       int    `json:"approximate_spend"`
}
