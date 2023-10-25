package model

import (
	"github.com/kaytu-io/kaytu-engine/pkg/analytics"
	"gorm.io/gorm"
)

type AnalyticsJob struct {
	gorm.Model
	ResourceCollectionId *string
	Status               analytics.JobStatus
	FailureMessage       string
}
