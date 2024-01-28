package types

import (
	complianceApi "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"time"
)

type ComplianceRunnerStatus string

const (
	ComplianceRunnerCreated    ComplianceRunnerStatus = "CREATED"
	ComplianceRunnerQueued     ComplianceRunnerStatus = "QUEUED"
	ComplianceRunnerInProgress ComplianceRunnerStatus = "IN_PROGRESS"
	ComplianceRunnerSucceeded  ComplianceRunnerStatus = "SUCCEEDED"
	ComplianceRunnerFailed     ComplianceRunnerStatus = "FAILED"
	ComplianceRunnerTimeOut    ComplianceRunnerStatus = "TIMEOUT"
)

type Job struct {
	ID          uint
	ParentJobID uint
	CreatedAt   time.Time

	ExecutionPlan ExecutionPlan
}

type ExecutionPlan struct {
	Callers []Caller
	Query   complianceApi.Query

	ConnectionID         *string
	ProviderConnectionID *string
}

type Caller struct {
	RootBenchmark      string
	ParentBenchmarkIDs []string
	ControlID          string
	ControlSeverity    types.FindingSeverity
}

type JobResult struct {
	Job               Job
	StartedAt         time.Time
	Status            ComplianceRunnerStatus
	Error             string
	TotalFindingCount *int
}
