package model

import (
	query_runner "github.com/kaytu-io/open-governance/pkg/inventory/query-runner"
	"gorm.io/gorm"
)

type QueryRunnerJob struct {
	gorm.Model
	RunId      uint
	RetryCount int
	Status     query_runner.QueryRunnerStatus
	CreatedBy  string
	QueryId    string
}
