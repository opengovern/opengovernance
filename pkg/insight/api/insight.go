package api

type InsightJobStatus string

const (
	InsightJobCreated    InsightJobStatus = "CREATED"
	InsightJobInProgress InsightJobStatus = "IN_PROGRESS"
	InsightJobFailed     InsightJobStatus = "FAILED"
	InsightJobSucceeded  InsightJobStatus = "SUCCEEDED"
)
