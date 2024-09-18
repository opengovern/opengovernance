package model

import (
	query_runner "github.com/kaytu-io/open-governance/pkg/inventory/query-runner"
	"gorm.io/gorm"
)

type QueryRunnerJob struct {
	gorm.Model
	QueryId            string
	CreatedBy          string
	RetryCount         int
	Status             query_runner.QueryRunnerStatus
	FailureMessage     string
	NatsSequenceNumber uint64
}
