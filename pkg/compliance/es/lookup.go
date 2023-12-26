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
	Aggregations struct {
		Resources struct {
			Buckets []struct {
				Key       string `json:"key"`
				HitSelect struct {
					Hits struct {
						Hits []struct {
							Source es.LookupResource `json:"_source"`
						} `json:"hits"`
					} `json:"hits"`
				} `json:"hit_select"`
			} `json:"buckets"`
		} `json:"resources"`
	} `json:"aggregations"`
}

func FetchLookupByResourceIDBatch(client kaytu.Client, resourceID []string) ([]es.LookupResource, error) {
	request := make(map[string]any)
	request["size"] = 0
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
	request["aggs"] = map[string]any{
		"resources": map[string]any{
			"terms": map[string]any{
				"field": "resource_id",
			},
			"aggs": map[string]any{
				"hit_select": map[string]any{
					"top_hits": map[string]any{
						"size": 1,
						"sort": map[string]any{
							"_id": "desc",
						},
					},
				},
			},
		},
	}

	b, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	fmt.Println("query=", string(b), "index=", InventorySummaryIndex)

	var response LookupQueryResponse
	err = client.Search(context.Background(), InventorySummaryIndex, string(b), &response)
	if err != nil {
		return nil, err
	}

	resources := make([]es.LookupResource, 0, len(response.Aggregations.Resources.Buckets))
	for _, bucket := range response.Aggregations.Resources.Buckets {
		if len(bucket.HitSelect.Hits.Hits) == 0 {
			continue
		}
		resources = append(resources, bucket.HitSelect.Hits.Hits[0].Source)
	}

	return resources, nil
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
