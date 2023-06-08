package api

import (
	"time"

	complianceapi "gitlab.com/keibiengine/keibi-engine/pkg/compliance/api"
)

type EvaluateStack struct {
	Benchmarks []string `json:"benchmarks" validate:"required"`
	StackID    string   `json:"stackId"`
}

type UpdateStackResourcesRequest struct {
	ResourcesToAdd    []string `json:"resourcesToAdd"`
	ResourcesToRemove []string `json:"resourcesToRemove"`
}

type Stack struct {
	StackID     string              `json:"stackId" validate:"required"`
	CreatedAt   time.Time           `json:"createdAt"`
	UpdatedAt   time.Time           `json:"updatedAt"`
	Resources   []string            `json:"resources"`
	Tags        map[string][]string `json:"tags"`
	Evaluations []StackEvaluation   `json:"evaluations,omitempty"`
	AccountIDs  []string            `json:"accountIds"`
}

type StackEvaluation struct {
	BenchmarkID string
	JobID       uint
	CreatedAt   time.Time
}

type GetStackFindings struct {
	Sorts []complianceapi.FindingSortItem `json:"sorts"`
	Page  complianceapi.Page              `json:"page" validate:"required"`
}
