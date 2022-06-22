package api

type InsightJobStatus string

const (
	InsightJobInProgress InsightJobStatus = "IN_PROGRESS"
	InsightJobFailed     InsightJobStatus = "FAILED"
	InsightJobSucceeded  InsightJobStatus = "SUCCEEDED"
)
