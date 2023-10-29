package api

type ComplianceReportJobStatus string

const (
	ComplianceReportJobCreated              ComplianceReportJobStatus = "CREATED"
	ComplianceReportJobInProgress           ComplianceReportJobStatus = "IN_PROGRESS"
	ComplianceReportJobCompletedWithFailure ComplianceReportJobStatus = "COMPLETED_WITH_FAILURE"
	ComplianceReportJobCompleted            ComplianceReportJobStatus = "COMPLETED"
)
