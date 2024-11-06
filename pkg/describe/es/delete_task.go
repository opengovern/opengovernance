package es

import (
	"github.com/opengovern/og-util/pkg/integration"
)

const DeleteTasksIndex = "delete_tasks"

type DeletingResource struct {
	Key        []byte
	ResourceID string
	Index      string
}

type DeleteTaskType string

const (
	DeleteTaskTypeResource DeleteTaskType = "resource"
	DeleteTaskTypeQuery    DeleteTaskType = "query"
)

type DeleteTask struct {
	EsID    string `json:"es_id"`
	EsIndex string `json:"es_index"`

	TaskType        DeleteTaskType   `json:"task_type"`
	DiscoveryJobID  uint             `json:"discovery_job_id"`
	IntegrationID   string           `json:"integration_id"`
	ResourceType    string           `json:"resource_type"`
	IntegrationType integration.Type `json:"connector"`

	DeletingResources []DeletingResource `json:"deleting_resources"`
	Query             string             `json:"query"`
	QueryIndex        string             `json:"query_index"`
}

func (d DeleteTask) KeysAndIndex() ([]string, string) {
	var ids []string
	for _, r := range d.DeletingResources {
		ids = append(ids, r.ResourceID)
		ids = append(ids, r.Index)
	}
	ids = append(ids, d.ResourceType)
	ids = append(ids, d.IntegrationID)
	ids = append(ids, string(d.TaskType))
	ids = append(ids, d.Query)
	return ids, DeleteTasksIndex
}
