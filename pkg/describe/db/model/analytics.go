package model

import (
	"github.com/kaytu-io/kaytu-engine/pkg/analytics"
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
	Status         analytics.JobStatus
	FailureMessage string
}
