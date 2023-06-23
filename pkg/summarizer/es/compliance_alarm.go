package es

import (
	"fmt"

	"github.com/kaytu-io/kaytu-util/pkg/source"

	"github.com/kaytu-io/kaytu-engine/pkg/types"
)

const (
	AlarmIndex = "finding_alarms"
)

type Event struct {
	ResourceID    string                 `json:"resource_id"`
	ControlID     string                 `json:"control_id"`
	CreatedAt     int64                  `json:"created_at"`
	ScheduleJobID uint                   `json:"schedule_job_id"`
	Status        types.ComplianceResult `json:"status"`
}

type FindingAlarm struct {
	ResourceID    string                 `json:"resource_id"`
	BenchmarkID   string                 `json:"benchmarkID"`
	ControlID     string                 `json:"control_id"`
	ResourceType  string                 `json:"resourceType"`
	SourceID      string                 `json:"sourceID"`
	SourceType    source.Type            `json:"sourceType"`
	Severity      types.Severity         `json:"policySeverity"`
	CreatedAt     int64                  `json:"created_at"`
	ScheduleJobID uint                   `json:"schedule_job_id"`
	LastEvaluated int64                  `json:"last_evaluated"`
	Status        types.ComplianceResult `json:"status"`
	Events        []Event                `json:"events"`
}

func (r FindingAlarm) KeysAndIndex() ([]string, string) {
	keys := []string{
		r.ResourceID,
		r.ControlID,
		fmt.Sprintf("%d", r.CreatedAt),
	}
	return keys, AlarmIndex
}
