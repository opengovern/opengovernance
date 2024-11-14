package runner

import "time"

type ComplianceRunnerStatus string

const (
	ComplianceRunnerCreated    ComplianceRunnerStatus = "CREATED"
	ComplianceRunnerQueued     ComplianceRunnerStatus = "QUEUED"
	ComplianceRunnerInProgress ComplianceRunnerStatus = "IN_PROGRESS"
	ComplianceRunnerSucceeded  ComplianceRunnerStatus = "SUCCEEDED"
	ComplianceRunnerFailed     ComplianceRunnerStatus = "FAILED"
	ComplianceRunnerTimeOut    ComplianceRunnerStatus = "TIMEOUT"
	ComplianceRunnerCanceled   ComplianceRunnerStatus = "CANCELED"
)

type JobResult struct {
	Job                        Job
	StartedAt                  time.Time
	Status                     ComplianceRunnerStatus
	Error                      string
	TotalComplianceResultCount *int
}
