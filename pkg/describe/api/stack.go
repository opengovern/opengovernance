package api

type EvaluateStack struct {
	Benchmarks []string `json:"benchmarks" validate:"required"`
	StackID    string   `json:"stackId"`
}

type CreateStackRequest struct {
	Statefile string     `json:"statefile"`
	Resources []string   `json:"resources"`
	Tags      []StackTag `json:"tags"`
}

type StackTag struct {
	Key   string   `json:"key"`
	Value []string `json:"value"`
}

type UpdateStackResourcesRequest struct {
	ResourcesToAdd    []string `json:"resourcesToAdd"`
	ResourcesToRemove []string `json:"resourcesToRemove"`
}
