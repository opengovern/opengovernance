package model

import (
	"encoding/json"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/runner/types"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/summarizer"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/api"
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

func (c ComplianceJobStatus) ToApi() api.ComplianceJobStatus {
	return api.ComplianceJobStatus(c)
}

type ComplianceJob struct {
	gorm.Model
	BenchmarkID    string
	Status         ComplianceJobStatus
	FailureMessage string
	IsStack        bool
}

func (c ComplianceJob) ToApi() api.ComplianceJob {
	return api.ComplianceJob{
		ID:             c.ID,
		BenchmarkID:    c.BenchmarkID,
		Status:         c.Status.ToApi(),
		FailureMessage: c.FailureMessage,
		IsStack:        c.IsStack,
	}
}

type ComplianceRunner struct {
	gorm.Model

	Callers              string
	BenchmarkID          string
	QueryID              string
	ConnectionID         *string
	ResourceCollectionID *string
	ParentJobID          uint `gorm:"index"`

	StartedAt         time.Time
	TotalFindingCount *int
	Status            types.ComplianceRunnerStatus
	FailureMessage    string
	RetryCount        int
}

func (cr *ComplianceRunner) GetCallers() ([]types.Caller, error) {
	var res []types.Caller
	err := json.Unmarshal([]byte(cr.Callers), &res)
	return res, err
}

func (cr *ComplianceRunner) SetCallers(callers []types.Caller) error {
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
