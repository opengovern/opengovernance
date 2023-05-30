package api

import "time"

type EvaluateStack struct {
	Benchmarks []string `json:"benchmarks" validate:"required"`
	StackID    string   `json:"stackId"`
}

type CreateStackRequest struct {
	Statefile string              `json:"statefile"`
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
}

type StackEvaluation struct {
	BenchmarkID string
	JobID       uint
	CreatedAt   time.Time
}
