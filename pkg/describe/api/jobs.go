package api

import (
	"time"

	"github.com/kaytu-io/kaytu-util/pkg/source"
	insightapi "gitlab.com/keibiengine/keibi-engine/pkg/insight/api"
)

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

type InsightJob struct {
	ID             uint                        `json:"id"`
	InsightID      uint                        `json:"insightId"`
	SourceID       string                      `json:"sourceId"`
	AccountID      string                      `json:"accountId"`
	SourceType     source.Type                 `json:"sourceType"`
	Status         insightapi.InsightJobStatus `json:"status"`
	FailureMessage string                      `json:"FailureMessage,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
