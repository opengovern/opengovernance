package query_runner

const (
	JobQueueTopic       = "query-runner-jobs-queue"
	JobResultQueueTopic = "query-runner-results-queue"
	ConsumerGroup       = "query-runner-worker"
	StreamName          = "query-runner-worker"
)
