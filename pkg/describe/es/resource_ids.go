package es

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-util/pkg/es"
	"strings"

	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
)

type ResourceIdentifierFetchResponse struct {
	Hits ResourceIdentifierFetchHits `json:"hits"`
}
type ResourceIdentifierFetchHits struct {
	Total kaytu.SearchTotal            `json:"total"`
	Hits  []ResourceIdentifierFetchHit `json:"hits"`
}
type ResourceIdentifierFetchHit struct {
	ID      string            `json:"_id"`
	Score   float64           `json:"_score"`
	Index   string            `json:"_index"`
	Type    string            `json:"_type"`
	Version int64             `json:"_version,omitempty"`
	Source  es.LookupResource `json:"_source"`
	Sort    []any             `json:"sort"`
}

func GetResourceIDsForAccountResourceTypeFromES(client kaytu.Client, sourceID, resourceType string, additionalFilters []map[string]any, searchAfter []any, size int) (*ResourceIdentifierFetchResponse, error) {
	root := map[string]any{}
	root["query"] = map[string]any{
		"bool": map[string]any{
			"filter": append([]map[string]any{
				{"term": map[string]string{"source_id": sourceID}},
				{"term": map[string]string{"resource_type": strings.ToLower(resourceType)}},
			}, additionalFilters...),
		},
	}
	if searchAfter != nil {
		root["search_after"] = searchAfter
	}
	root["size"] = size
	root["sort"] = []map[string]any{
		{"created_at": "asc"},
		{"_id": "desc"},
	}

	queryBytes, err := json.Marshal(root)
	if err != nil {
		return nil, err
	}

	var response ResourceIdentifierFetchResponse
	err = client.Search(context.Background(), es.InventorySummaryIndex,
		string(queryBytes), &response)
	if err != nil {
		fmt.Println("query=", string(queryBytes))
		return nil, err
	}

	return &response, nil
}

func GetResourceIDsForAccountFromES(client kaytu.Client, sourceID string, searchAfter []any, size int) (*ResourceIdentifierFetchResponse, error) {
	root := map[string]any{}
	root["query"] = map[string]any{
		"bool": map[string]any{
			"filter": []map[string]any{
				{"term": map[string]string{"source_id": sourceID}},
			},
		},
	}
	if searchAfter != nil {
		root["search_after"] = searchAfter
	}
	root["size"] = size
	root["sort"] = []map[string]any{
		{"created_at": "asc"},
		{"_id": "desc"},
	}

	queryBytes, err := json.Marshal(root)
	if err != nil {
		return nil, err
	}

	var response ResourceIdentifierFetchResponse
	err = client.Search(context.Background(), es.InventorySummaryIndex,
		string(queryBytes), &response)
	if err != nil {
		fmt.Println("query=", string(queryBytes))
		return nil, err
	}

	return &response, nil
}

type InventoryCountResponse struct {
	Took     int  `json:"took"`
	TimedOut bool `json:"timed_out"`
	Shards   struct {
		Total      int `json:"total"`
		Successful int `json:"successful"`
		Skipped    int `json:"skipped"`
		Failed     int `json:"failed"`
	} `json:"_shards"`
	Hits struct {
		Total struct {
			Value    int64  `json:"value"`
			Relation string `json:"relation"`
		} `json:"total"`
		MaxScore interface{}   `json:"max_score"`
		Hits     []interface{} `json:"hits"`
	} `json:"hits"`
}

func GetInventoryCountResponse(client kaytu.Client) (int64, error) {
	root := map[string]any{}
	root["size"] = 0

	queryBytes, err := json.Marshal(root)
	if err != nil {
		return 0, err
	}

	var response InventoryCountResponse
	err = client.Search(context.Background(), es.InventorySummaryIndex,
		string(queryBytes), &response)
	if err != nil {
		fmt.Println("query=", string(queryBytes))
		return 0, err
	}

	return response.Hits.Total.Value, nil
}
