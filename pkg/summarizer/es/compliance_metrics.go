package es

import (
	"fmt"
	"time"
)

const (
	MetricsIndex = "finding_metrics"
)

type FindingMetrics struct {
	ScheduleJobID        uint      `json:"schedule_job_id"`
	PassedFindingsCount  int64     `json:"passed_findings_count"`
	FailedFindingsCount  int64     `json:"failed_findings_count"`
	UnknownFindingsCount int64     `json:"unknown_findings_count"`
	DescribedAt          time.Time `json:"described_at"`
	EvaluatedAt          time.Time `json:"evaluated_at"`
}

func (r FindingMetrics) KeysAndIndex() ([]string, string) {
	keys := []string{
		fmt.Sprintf("%d", r.ScheduleJobID),
	}
	return keys, MetricsIndex
}
