package model

import (
	"encoding/json"
	api2 "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/runner"
	"github.com/lib/pq"
	"gorm.io/gorm"
	"time"
)

type ComplianceJob struct {
	gorm.Model
	BenchmarkID    string
	RunnerIDs      pq.Int64Array `gorm:"type:bigint[]"`
	Status         api2.ComplianceReportJobStatus
	FailureMessage string
	IsStack        bool
}

type ComplianceRunner struct {
	gorm.Model

	Callers              string
	BenchmarkID          string
	QueryID              string
	ConnectionID         *string
	ResourceCollectionID *string

	StartedAt      time.Time
	RetryCount     int
	Status         runner.ComplianceRunnerStatus
	FailureMessage string
}

func (cr *ComplianceRunner) GetCallers() ([]runner.Caller, error) {
	var res []runner.Caller
	err := json.Unmarshal([]byte(cr.Callers), &res)
	return res, err
}

func (cr *ComplianceRunner) SetCallers(callers []runner.Caller) error {
	b, err := json.Marshal(callers)
	if err != nil {
		return err
	}
	cr.Callers = string(b)
	return nil
}
