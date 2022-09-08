package api

type SummarizerJobStatus string

const (
	SummarizerJobInProgress SummarizerJobStatus = "IN_PROGRESS"
	SummarizerJobFailed     SummarizerJobStatus = "FAILED"
	SummarizerJobSucceeded  SummarizerJobStatus = "SUCCEEDED"
)
