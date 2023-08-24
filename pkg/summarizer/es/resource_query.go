package es

import (
	"context"
	"encoding/json"

	"github.com/kaytu-io/kaytu-engine/pkg/describe/es"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
)

const (
	EsFetchPageSize       = 10000
	EsTermSize            = 10000
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

func FetchLookupByResourceTypes(client kaytu.Client, resourceTypes []string, searchAfter []any, size int) (LookupQueryResponse, error) {
	res := make(map[string]any)
	resourceTypes = utils.ToLowerStringSlice(resourceTypes)
	res["query"] = map[string]any{
		"bool": map[string]any{
			"filter": []any{
				map[string]any{
					"terms": map[string][]string{"resource_type": resourceTypes},
				},
			},
		},
	}

	if searchAfter != nil {
		res["search_after"] = searchAfter
	}

	res["size"] = size
	res["sort"] = []map[string]any{
		{"created_at": "asc"},
		{"_id": "desc"},
	}
	b, err := json.Marshal(res)
	if err != nil {
		return LookupQueryResponse{}, err
	}

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
		} `json:"hits"`
	} `json:"hits"`
}

func FetchResourcesByResourceTypes(client kaytu.Client, resourceType string, searchAfter []any, size int) (ResourceQueryResponse, error) {
	res := make(map[string]any)
	res["query"] = map[string]any{
		"bool": map[string]any{
			"filter": []any{
				map[string]any{
					"term": map[string]string{"resource_type": resourceType},
				},
			},
		},
	}

	if searchAfter != nil {
		res["search_after"] = searchAfter
	}

	res["size"] = size
	res["sort"] = []map[string]any{
		{"created_at": "asc"},
		{"_id": "desc"},
	}
	b, err := json.Marshal(res)
	if err != nil {
		return ResourceQueryResponse{}, err
	}

	var response ResourceQueryResponse
	err = client.Search(context.Background(), es.ResourceTypeToESIndex(resourceType), string(b), &response)
	if err != nil {
		return ResourceQueryResponse{}, err
	}

	return response, nil
}
