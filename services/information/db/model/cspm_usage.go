package model

import (
	"github.com/lib/pq"
	"gorm.io/gorm"
	"time"
)

type CspmUsage struct {
	gorm.Model

	WorkspaceId               string         `json:"workspace_id" gorm:"index"`
	AwsOrganizationRootEmails pq.StringArray `gorm:"type:citext[]"`
	AwsAccountCount           int            `json:"aws_account_count"`
	AzureAdPrimaryDomains     pq.StringArray `gorm:"type:citext[]"`
	AzureSubscriptionCount    int            `json:"azure_account_count"`
	Users                     pq.StringArray `gorm:"type:citext[]"`
	GatherTimestamp           time.Time      `json:"gather_timestamp"`
}
