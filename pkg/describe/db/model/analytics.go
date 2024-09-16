package model

import (
	"github.com/kaytu-io/open-governance/pkg/analytics/api"
	"gorm.io/gorm"
)

type AnalyticsJobType string

const (
	AnalyticsJobTypeNormal             AnalyticsJobType = "normal"
	AnalyticsJobTypeResourceCollection AnalyticsJobType = "resource_collection"
)

type AnalyticsJob struct {
	gorm.Model
	Type           AnalyticsJobType
	Status         api.JobStatus
	FailureMessage string
}
