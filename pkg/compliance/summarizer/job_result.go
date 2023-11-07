package summarizer

import "time"

type ComplianceSummarizerStatus string

const (
	ComplianceSummarizerCreated    ComplianceSummarizerStatus = "CREATED"
	ComplianceSummarizerInProgress ComplianceSummarizerStatus = "IN_PROGRESS"
	ComplianceSummarizerSucceeded  ComplianceSummarizerStatus = "SUCCEEDED"
	ComplianceSummarizerFailed     ComplianceSummarizerStatus = "FAILED"
)

type JobResult struct {
	Job       Job
	StartedAt time.Time
	Status    ComplianceSummarizerStatus
	Error     string
}
