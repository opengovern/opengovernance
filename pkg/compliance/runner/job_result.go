package runner

import "time"

type ComplianceRunnerStatus string

const (
	ComplianceRunnerCreated    ComplianceRunnerStatus = "CREATED"
	ComplianceRunnerInProgress ComplianceRunnerStatus = "IN_PROGRESS"
	ComplianceRunnerSucceeded  ComplianceRunnerStatus = "SUCCEEDED"
	ComplianceRunnerFailed     ComplianceRunnerStatus = "FAILED"
)

type JobResult struct {
	Job               Job
	StartedAt         time.Time
	Status            ComplianceRunnerStatus
	Error             string
	TotalFindingCount *int
}
