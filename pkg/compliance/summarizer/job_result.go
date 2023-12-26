package summarizer

import (
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/summarizer/types"
	"time"
)

type ComplianceSummarizerStatus string

const (
	ComplianceSummarizerCreated    ComplianceSummarizerStatus = "CREATED"
	ComplianceSummarizerInProgress ComplianceSummarizerStatus = "IN_PROGRESS"
	ComplianceSummarizerSucceeded  ComplianceSummarizerStatus = "SUCCEEDED"
	ComplianceSummarizerFailed     ComplianceSummarizerStatus = "FAILED"
)

type JobResult struct {
	Job       types.Job
	StartedAt time.Time
	Status    ComplianceSummarizerStatus
	Error     string
}
