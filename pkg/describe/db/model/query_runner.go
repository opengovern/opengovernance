package model

import (
	queryrunner "github.com/opengovern/opengovernance/services/inventory/query-runner"
	"gorm.io/gorm"
)

type QueryRunnerJob struct {
	gorm.Model
	QueryId            string
	CreatedBy          string
	RetryCount         int
	Status             queryrunner.QueryRunnerStatus
	FailureMessage     string
	NatsSequenceNumber uint64
}
