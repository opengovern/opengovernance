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
	ID              uint                      `json:"id" example:"1"`
	UpdatedAt       time.Time                 `json:"updatedAt" example:"2021-01-01T00:00:00Z"`
	ReportCreatedAt int64                     `json:"reportCreatedAt" example:"1619510400"`
	Status          ComplianceReportJobStatus `json:"status" example:"InProgress"`
	FailureMessage  string                    `json:"failureMessage" example:""`
}
