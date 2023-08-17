package es

import (
	"context"
	"encoding/json"
	"fmt"
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
	ID      string         `json:"_id"`
	Score   float64        `json:"_score"`
	Index   string         `json:"_index"`
	Type    string         `json:"_type"`
	Version int64          `json:"_version,omitempty"`
	Source  LookupResource `json:"_source"`
	Sort    []any          `json:"sort"`
}

func GetResourceIDsForAccountResourceTypeFromES(client kaytu.Client, sourceID, resourceType string, searchAfter []any, size int) (*ResourceIdentifierFetchResponse, error) {
	root := map[string]any{}
	root["query"] = map[string]any{
		"bool": map[string]any{
			"filter": []map[string]any{
				{"term": map[string]string{"source_id": sourceID}},
				{"term": map[string]string{"resource_type": strings.ToLower(resourceType)}},
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
	err = client.Search(context.Background(), InventorySummaryIndex,
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
	err = client.Search(context.Background(), InventorySummaryIndex,
		string(queryBytes), &response)
	if err != nil {
		fmt.Println("query=", string(queryBytes))
		return nil, err
	}

	return &response, nil
}
