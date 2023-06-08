package api

import "github.com/kaytu-io/kaytu-util/pkg/source"

type GetCredsForJobRequest struct {
	SourceID string `json:"sourceId"`
}

type GetCredsForJobResponse struct {
	Credentials string `json:"creds"`
}

type GetDataResponse struct {
	Data string `json:"data"`
}

type TriggerBenchmarkEvaluationRequest struct {
	BenchmarkID  string   `json:"benchmarkID"`
	ConnectionID *string  `json:"connectionID"`
	ResourceIDs  []string `json:"resourceIDs"`
}

type TriggerInsightEvaluationRequest struct {
	InsightID    uint     `json:"insightID"`
	ConnectionID *string  `json:"connectionID"`
	ResourceIDs  []string `json:"resourceIDs"`
}

type ListBenchmarkEvaluationsRequest struct {
	EvaluatedAtAfter  *int64       `json:"evaluatedAtAfter"`
	EvaluatedAtBefore *int64       `json:"evaluatedAtBefore"`
	ConnectionID      *string      `json:"connectionID"`
	Connector         *source.Type `json:"connector"`
	BenchmarkID       *string      `json:"benchmarkID"`
}
