package es

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

const DeleteTasksIndex = "delete_tasks"

type DeletingResource struct {
	Key        []byte
	ResourceID string
	Index      string
}

type DeleteTask struct {
	EsID    string `json:"es_id"`
	EsIndex string `json:"es_index"`

	DeletingResources []DeletingResource `json:"deleting_resources"`
	DiscoveryJobID    uint               `json:"discovery_job_id"`
	ConnectionID      string             `json:"connection_id"`
	ResourceType      string             `json:"resource_type"`
	Connector         source.Type        `json:"connector"`
}

func (d DeleteTask) KeysAndIndex() ([]string, string) {
	var ids []string
	for _, r := range d.DeletingResources {
		ids = append(ids, r.ResourceID)
		ids = append(ids, r.Index)
	}
	ids = append(ids, d.ResourceType)
	ids = append(ids, d.ConnectionID)
	return ids, DeleteTasksIndex
}

type GetDeleteTasksResponse struct {
	Hits GetDeleteTasksHits `json:"hits"`
}
type GetDeleteTasksHits struct {
	Total kaytu.SearchTotal   `json:"total"`
	Hits  []GetDeleteTasksHit `json:"hits"`
}
type GetDeleteTasksHit struct {
	ID      string     `json:"_id"`
	Score   float64    `json:"_score"`
	Index   string     `json:"_index"`
	Type    string     `json:"_type"`
	Version int64      `json:"_version,omitempty"`
	Source  DeleteTask `json:"_source"`
	Sort    []any      `json:"sort"`
}

func GetDeleteTasks(client kaytu.Client, jobID uint) (*GetDeleteTasksResponse, error) {
	root := map[string]any{}
	root["query"] = map[string]any{
		"bool": map[string]any{
			"filter": append([]map[string]any{
				{"term": map[string]any{"discovery_job_id": jobID}},
			}),
		},
	}
	root["size"] = 10000
	root["sort"] = []map[string]any{
		{"_id": "desc"},
	}

	queryBytes, err := json.Marshal(root)
	if err != nil {
		return nil, err
	}

	var response GetDeleteTasksResponse
	err = client.Search(context.Background(), DeleteTasksIndex,
		string(queryBytes), &response)
	if err != nil {
		fmt.Println("query=", string(queryBytes))
		return nil, err
	}

	return &response, nil
}
