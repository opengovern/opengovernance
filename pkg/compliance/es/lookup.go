package es

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-util/pkg/es"

	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
)

const (
	InventorySummaryIndex = "inventory_summary"
)

type LookupQueryResponse struct {
	Hits struct {
		Total kaytu.SearchTotal `json:"total"`
		Hits  []struct {
			ID      string            `json:"_id"`
			Score   float64           `json:"_score"`
			Index   string            `json:"_index"`
			Type    string            `json:"_type"`
			Version int64             `json:"_version,omitempty"`
			Source  es.LookupResource `json:"_source"`
			Sort    []any             `json:"sort"`
		} `json:"hits"`
	} `json:"hits"`
}

func FetchLookupsByResourceIDWildcard(client kaytu.Client, resourceID string) (LookupQueryResponse, error) {
	request := make(map[string]any)
	request["size"] = 1
	request["sort"] = []map[string]any{
		{
			"_id": "desc",
		},
	}
	request["query"] = map[string]any{
		"bool": map[string]any{
			"filter": map[string]any{
				"term": map[string]any{
					"resource_id": resourceID,
				},
			},
		},
	}
	b, err := json.Marshal(request)
	if err != nil {
		return LookupQueryResponse{}, err
	}

	fmt.Println("query=", string(b), "index=", InventorySummaryIndex)

	var response LookupQueryResponse
	err = client.Search(context.Background(), InventorySummaryIndex, string(b), &response)
	if err != nil {
		return LookupQueryResponse{}, err
	}

	if len(response.Hits.Hits) > 0 {
		return response, nil
	}

	request["query"] = map[string]any{
		"bool": map[string]any{
			"filter": map[string]any{
				"regexp": map[string]any{
					"resource_id": map[string]any{
						"value":            fmt.Sprintf(".*%s.*", resourceID),
						"case_insensitive": true,
					},
				},
			},
		},
	}

	b, err = json.Marshal(request)
	if err != nil {
		return LookupQueryResponse{}, err
	}
	fmt.Println("query=", string(b), "index=", InventorySummaryIndex)
	response = LookupQueryResponse{}
	err = client.Search(context.Background(), InventorySummaryIndex, string(b), &response)
	if err != nil {
		return LookupQueryResponse{}, err
	}

	return response, nil
}

func FetchLookupByResourceIDBatch(client kaytu.Client, resourceID []string) (LookupQueryResponse, error) {
	request := make(map[string]any)
	request["size"] = len(resourceID)
	request["sort"] = []map[string]any{
		{
			"_id": "desc",
		},
	}
	request["query"] = map[string]any{
		"bool": map[string]any{
			"filter": map[string]any{
				"terms": map[string]any{
					"resource_id": resourceID,
				},
			},
		},
	}
	b, err := json.Marshal(request)
	if err != nil {
		return LookupQueryResponse{}, err
	}

	fmt.Println("query=", string(b), "index=", InventorySummaryIndex)

	var response LookupQueryResponse
	err = client.Search(context.Background(), InventorySummaryIndex, string(b), &response)
	if err != nil {
		return LookupQueryResponse{}, err
	}

	return response, nil
}

type ResourceQueryResponse struct {
	Hits struct {
		Total kaytu.SearchTotal `json:"total"`
		Hits  []struct {
			ID      string      `json:"_id"`
			Score   float64     `json:"_score"`
			Index   string      `json:"_index"`
			Type    string      `json:"_type"`
			Version int64       `json:"_version,omitempty"`
			Source  es.Resource `json:"_source"`
			Sort    []any       `json:"sort"`
		}
	}
}

func FetchResourceByResourceIdAndType(client kaytu.Client, resourceId string, resourceType string) (*es.Resource, error) {
	request := make(map[string]any)
	request["size"] = 1
	request["query"] = map[string]any{
		"bool": map[string]any{
			"filter": map[string]any{
				"term": map[string]any{
					"id": resourceId,
				},
			},
		},
	}
	b, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	index := es.ResourceTypeToESIndex(resourceType)

	fmt.Println("query=", string(b), "index=", index)

	var response ResourceQueryResponse
	err = client.Search(context.Background(), index, string(b), &response)
	if err != nil {
		return nil, err
	}

	if len(response.Hits.Hits) == 0 {
		return nil, nil
	}

	return &response.Hits.Hits[0].Source, nil
}
