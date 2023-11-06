package model

import (
	"encoding/json"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/runner"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/summarizer"
	"gorm.io/gorm"
	"time"
)

type ComplianceJobStatus string

const (
	ComplianceJobCreated              ComplianceJobStatus = "CREATED"
	ComplianceJobRunnersInProgress    ComplianceJobStatus = "RUNNERS_IN_PROGRESS"
	ComplianceJobSummarizerInProgress ComplianceJobStatus = "SUMMARIZER_IN_PROGRESS"
	ComplianceJobFailed               ComplianceJobStatus = "FAILED"
	ComplianceJobSucceeded            ComplianceJobStatus = "SUCCEEDED"
)

type ComplianceJob struct {
	gorm.Model
	BenchmarkID    string
	Status         ComplianceJobStatus
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
	ParentJobID          uint

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

type ComplianceSummarizer struct {
	gorm.Model

	BenchmarkID string
	ParentJobID uint

	StartedAt      time.Time
	RetryCount     int
	Status         summarizer.ComplianceSummarizerStatus
	FailureMessage string
}
