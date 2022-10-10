package es

import (
	"fmt"

	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"

	"gitlab.com/keibiengine/keibi-engine/pkg/types"
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
	ResourceID     string                 `json:"resource_id"`
	BenchmarkID    string                 `json:"benchmarkID"`
	ControlID      string                 `json:"control_id"`
	ResourceType   string                 `json:"resourceType"`
	ServiceName    string                 `json:"serviceName"`
	SourceID       uuid.UUID              `json:"sourceID"`
	SourceType     source.Type            `json:"sourceType"`
	PolicySeverity string                 `json:"policySeverity"`
	CreatedAt      int64                  `json:"created_at"`
	ScheduleJobID  uint                   `json:"schedule_job_id"`
	LastEvaluated  int64                  `json:"last_evaluated"`
	Status         types.ComplianceResult `json:"status"`
	Events         []Event                `json:"events"`
}

func (r FindingAlarm) KeysAndIndex() ([]string, string) {
	keys := []string{
		r.ResourceID,
		r.ControlID,
		fmt.Sprintf("%d", r.CreatedAt),
	}
	return keys, AlarmIndex
}
