package api

import (
	"github.com/opengovern/opencomply/pkg/types"
)

type AuditSummary struct {
	Controls     map[string]types.AuditControlResult `json:"controls"`
	AuditSummary map[types.ComplianceStatus]uint64   `json:"audit_summary"`
	JobSummary   types.JobSummary                    `json:"job_summary"`
}
