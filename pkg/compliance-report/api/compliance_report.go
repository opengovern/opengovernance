package api

import "time"

type ComplianceReportJobStatus string

const (
	ComplianceReportJobCreated              ComplianceReportJobStatus = "CREATED"
	ComplianceReportJobInProgress           ComplianceReportJobStatus = "IN_PROGRESS"
	ComplianceReportJobCompletedWithFailure ComplianceReportJobStatus = "COMPLETED_WITH_FAILURE"
	ComplianceReportJobCompleted            ComplianceReportJobStatus = "COMPLETED"
)

type ComplianceReport struct {
	ID              uint                      `json:"id"`
	UpdatedAt       time.Time                 `json:"updatedAt"`
	ReportCreatedAt int64                     `json:"reportCreatedAt"`
	Status          ComplianceReportJobStatus `json:"status"`
	FailureMessage  string                    `json:"failureMessage"`
}
