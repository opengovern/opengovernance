package api

import (
	"time"

	complianceapi "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type StackStatus string

const (
	StackStatusPending              StackStatus = "PENDING"
	StackStatusStalled              StackStatus = "STALLED"
	StackStatusCreated              StackStatus = "CREATED"
	StackStatusDescribing           StackStatus = "DESCRIBING"
	StackStatusDescribed            StackStatus = "DESCRIBED_RESOURCES"
	StackStatusEvaluating           StackStatus = "EVALUATING"
	StackStatusFailed               StackStatus = "FAILED"
	StackStatusCompleted            StackStatus = "COMPLETED"
	StackStatusCompletedWithFailure StackStatus = "COMPLETED_WITH_FAILURE"
)

type EvaluationType string

const (
	EvaluationTypeInsight   EvaluationType = "INSIGHT"
	EvaluationTypeBenchmark EvaluationType = "BENCHMARK"
)

type StackEvaluationStatus string

const (
	StackEvaluationStatusInProgress StackEvaluationStatus = "IN_PROGRESS"
	StackEvaluationStatusFailed     StackEvaluationStatus = "COMPLETED_WITH_FAILURE"
	StackEvaluationStatusCompleted  StackEvaluationStatus = "COMPLETED"
)

type StackBenchmarkRequest struct {
	Benchmarks []string `json:"benchmarks" validate:"required" example:"[azure_cis_v140, azure_cis_v140_1, azure_cis_v140_1_1]"` // Benchmarks to add to the stack
	StackID    string   `json:"stackId" validate:"required" example:"stack-twr32a5d-5as5-4ffe-b1cc-e32w1ast87s0"`                // Stack unique identifier
}

type StackInsightRequest struct {
	Insights []uint `json:"insights" validate:"required" example:"[1, 2, 3]"`                                 // Insights to add to the stack
	StackID  string `json:"stackId" validate:"required" example:"stack-twr32a5d-5as5-4ffe-b1cc-e32w1ast87s0"` // Stack unique identifier
}

type UpdateStackResourcesRequest struct {
	ResourcesToAdd    []string `json:"resourcesToAdd"`
	ResourcesToRemove []string `json:"resourcesToRemove"`
}

type Stack struct {
	StackID        string              `json:"stackId" validate:"required" example:"stack-twr32a5d-5as5-4ffe-b1cc-e32w1ast87s0"`                              // Stack unique identifier
	CreatedAt      time.Time           `json:"createdAt" example:"2023-06-01T17:00:00.000000Z"`                                                               // Stack creation date
	UpdatedAt      time.Time           `json:"updatedAt" example:"2023-06-01T17:00:00.000000Z"`                                                               // Stack last update date
	Resources      []string            `json:"resources" example:"[/subscriptions/123/resourceGroups/rg-1/providers/Microsoft.Compute/virtualMachines/vm-1]"` // Stack resources list
	ResourceTypes  []string            `json:"resourceTypes" example:"[Microsoft.Compute/virtualMachines]"`                                                   // Stack resource types
	Tags           map[string][]string `json:"tags"`                                                                                                          // Stack tags
	Evaluations    []StackEvaluation   `json:"evaluations,omitempty"`                                                                                         // Stack evaluations history, including insight evaluations and compliance evaluations
	AccountIDs     []string            `json:"accountIds" example:"[0123456789]"`                                                                             // Accounts included in the stack
	SourceType     source.Type         `json:"sourceType" example:"Azure"`                                                                                    // Source type
	Status         StackStatus         `json:"status" example:"CREATED"`                                                                                      // Stack status. CREATED, EVALUATED, IN_PROGRESS, FAILED
	FailureMessage string              `json:"failureMessage,omitempty" example:"error message"`                                                              // Stack failure message
}

type StackEvaluation struct {
	EvaluatorID string         `json:"evaluatorId" example:"azure_cis_v140"`     // Benchmark ID or Insight ID
	Type        EvaluationType `json:"type" example:"BENCHMARK"`                 // BENCHMARK or INSIGHT
	JobID       uint           `json:"jobId" example:"1"`                        // Evaluation Job ID to find the job results
	CreatedAt   time.Time      `json:"createdAt" example:"2020-01-01T00:00:00Z"` // Evaluation creation date
	Status      StackEvaluationStatus
}

type GetStackFindings struct {
	BenchmarkIDs []string                        `json:"benchmarkIds" example:"azure_cis_v140"` // Benchmark IDs to filter
	Sorts        []complianceapi.FindingSortItem `json:"sorts"`                                 // Sorts to apply
	Page         complianceapi.Page              `json:"page" validate:"required"`              // Pages count to retrieve
}

type DescribeStackRequest struct {
	StackID string `json:"stackId" example:"stack-twr32a5d-5as5-4ffe-b1cc-e32w1ast87s0"`
	Config  any    `json:"config"`
}
