package model

import (
	api2 "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	"gorm.io/gorm"
)

type ComplianceJob struct {
	gorm.Model
	BenchmarkID        string
	Status             api2.ComplianceReportJobStatus
	FailureMessage     string
	IsStack            bool
	ResourceCollection *string
}
