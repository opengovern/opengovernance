package api

import (
	"time"

	complianceapi "gitlab.com/keibiengine/keibi-engine/pkg/compliance/api"
)

type EvaluateStack struct {
	Benchmarks []string `json:"benchmarks" validate:"required"`
	StackID    string   `json:"stackId"`
}

type CreateStackRequest struct {
	Resources []string            `json:"resources"`
	Tags      map[string][]string `json:"tags"`
}

type UpdateStackResourcesRequest struct {
	ResourcesToAdd    []string `json:"resourcesToAdd"`
	ResourcesToRemove []string `json:"resourcesToRemove"`
}

type Stack struct {
	StackID     string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Resources   []string
	Tags        map[string][]string
	Evaluations []StackEvaluation
	AccountIDs  []string
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
