package model

import (
	queryrunner "github.com/kaytu-io/open-governance/pkg/inventory/query-runner"
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
