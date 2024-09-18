package query_runner

type QueryRunnerStatus string

const (
	QueryRunnerCreated    QueryRunnerStatus = "CREATED"
	QueryRunnerQueued     QueryRunnerStatus = "QUEUED"
	QueryRunnerInProgress QueryRunnerStatus = "IN_PROGRESS"
	QueryRunnerSucceeded  QueryRunnerStatus = "SUCCEEDED"
	QueryRunnerFailed     QueryRunnerStatus = "FAILED"
	QueryRunnerTimeOut    QueryRunnerStatus = "TIMEOUT"
	QueryRunnerCanceled   QueryRunnerStatus = "CANCELED"
)

type JobResult struct {
	RunId          uint              `json:"runID"`
	Status         QueryRunnerStatus `json:"status"`
	FailureMessage string            `json:"failureMessage"`
}
