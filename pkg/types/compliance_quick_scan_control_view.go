package types

import (
	"strconv"
	"time"
)

type ComplianceQuickScanControlView struct {
	EsID    string `json:"es_id"`
	EsIndex string `json:"es_index"`

	Controls     map[string]AuditControlResult `json:"controls"`
	AuditSummary map[ComplianceStatus]uint64   `json:"audit_summary"`
	JobSummary   JobSummary                    `json:"job_summary"`
}

func (r ComplianceQuickScanControlView) KeysAndIndex() ([]string, string) {
	return []string{
		strconv.Itoa(int(r.JobSummary.JobID)),
	}, ComplianceQuickScanControlViewIndex
}

type AuditResourceFinding struct {
	ResourceID   string `json:"resource_id"`
	ResourceType string `json:"resource_type"`
	Reason       string `json:"reason"`
}

type AuditControlResult struct {
	Severity       ComplianceResultSeverity                    `json:"severity"`
	ControlSummary map[ComplianceStatus]uint64                 `json:"control_summary"`
	Results        map[ComplianceStatus][]AuditResourceFinding `json:"results"`
}

type JobSummary struct {
	JobID          uint      `json:"job_id"`
	FrameworkID    string    `json:"framework_id"`
	JobStartedAt   time.Time `json:"job_started_at"`
	IntegrationIDs []string  `json:"integration_ids"`
}
