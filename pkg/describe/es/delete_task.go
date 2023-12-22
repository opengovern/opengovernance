package es

import "github.com/kaytu-io/kaytu-util/pkg/source"

const deleteTasksIndex = "delete_tasks"

type DeletingResource struct {
	Key        []byte
	ResourceID string
	Index      string
}

type DeleteTask struct {
	EsID    string `json:"es_id"`
	EsIndex string `json:"es_index"`

	DeletingResources []DeletingResource
	ConnectionID      string
	ResourceType      string
	Connector         source.Type
}

func (d DeleteTask) KeysAndIndex() ([]string, string) {
	var ids []string
	for _, r := range d.DeletingResources {
		ids = append(ids, r.ResourceID)
		ids = append(ids, r.Index)
	}
	ids = append(ids, d.ResourceType)
	ids = append(ids, d.ConnectionID)
	return ids, deleteTasksIndex
}
