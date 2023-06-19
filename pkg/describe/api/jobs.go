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
	BenchmarkID  string   `json:"benchmarkID" example:"azure_cis_v1"`                                                                          // Benchmark ID to evaluate
	ConnectionID *string  `json:"connectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`                                                 // Connection ID to evaluate
	ResourceIDs  []string `json:"resourceIDs" example:"/subscriptions/123/resourceGroups/rg1/providers/Microsoft.Compute/virtualMachines/vm1"` // Resource IDs to evaluate
}

type TriggerInsightEvaluationRequest struct {
	InsightID    uint     `json:"insightID" example:"1"`                                                                                       // Insight ID to evaluate
	ConnectionID *string  `json:"connectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"`                                                 // Connection ID to evaluate
	ResourceIDs  []string `json:"resourceIDs" example:"/subscriptions/123/resourceGroups/rg1/providers/Microsoft.Compute/virtualMachines/vm1"` // Resource IDs to evaluate
}

type ListBenchmarkEvaluationsRequest struct {
	EvaluatedAtAfter  *int64       `json:"evaluatedAtAfter" example:"1619510400"`                       // Filter evaluations created after this timestamp
	EvaluatedAtBefore *int64       `json:"evaluatedAtBefore" example:"1619610400"`                      // Filter evaluations created before this timestamp
	ConnectionID      *string      `json:"connectionID" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"` // Filter evaluations for this connection
	Connector         *source.Type `json:"connector" example:"Azure"`                                   // Filter evaluations for this connector
	BenchmarkID       *string      `json:"benchmarkID" example:"azure_cis_v1"`                          // Filter evaluations for this benchmark
}

type InsightJob struct {
	ID             uint                        `json:"id" example:"1" format:"int64"`                           // Insight Job Unique ID
	InsightID      uint                        `json:"insightId" example:"1" format:"int64"`                    // Insight ID
	SourceID       string                      `json:"sourceId" example:"8e0f8e7a-1b1c-4e6f-b7e4-9c6af9d2b1c8"` // Source ID
	AccountID      string                      `json:"accountId" example:"0123456789"`                          // Account ID
	SourceType     source.Type                 `json:"sourceType" example:"Azure"`                              // Cloud provider
	Status         insightapi.InsightJobStatus `json:"status" example:"InProgress"`                             // Insight Job Status
	FailureMessage string                      `json:"FailureMessage,omitempty" example:""`                     // Failure Message

	CreatedAt time.Time `json:"createdAt" example:"2021-04-27T15:04:05Z"` // Insight Job creation timestamp
	UpdatedAt time.Time `json:"updatedAt" example:"2021-04-27T15:04:05Z"` // Insight Job last update timestamp
}
