package es

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
)

type ResourceIdentifierFetchResponse struct {
	Hits ResourceIdentifierFetchHits `json:"hits"`
}
type ResourceIdentifierFetchHits struct {
	Total keibi.SearchTotal            `json:"total"`
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

func GetResourceIDsForAccountResourceTypeFromES(client keibi.Client, sourceID, resourceType string, searchAfter []any, size int) (*ResourceIdentifierFetchResponse, error) {
	terms := map[string][]string{
		"source_id":     {sourceID},
		"resource_type": {strings.ToLower(resourceType)},
	}

	root := map[string]any{}
	if searchAfter != nil {
		root["search_after"] = searchAfter
	}
	root["size"] = size

	root["sort"] = []map[string]any{
		{
			"_id": "desc",
		},
	}

	boolQuery := make(map[string]any)
	var filters []map[string]any
	for k, vs := range terms {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{
				k: vs,
			},
		})
	}
	boolQuery["filter"] = filters
	root["query"] = map[string]any{
		"bool": boolQuery,
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

func GetResourceIDsForAccountFromES(client keibi.Client, sourceID string, searchAfter []any, size int) (*ResourceIdentifierFetchResponse, error) {
	terms := map[string][]string{
		"source_id": {sourceID},
	}

	root := map[string]any{}
	if searchAfter != nil {
		root["search_after"] = searchAfter
	}
	root["size"] = size

	root["sort"] = []map[string]any{
		{
			"_id": "desc",
		},
	}

	boolQuery := make(map[string]any)
	var filters []map[string]any
	for k, vs := range terms {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{
				k: vs,
			},
		})
	}
	boolQuery["filter"] = filters
	root["query"] = map[string]any{
		"bool": boolQuery,
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

func DeleteByIds(client keibi.Client, index string, ids []string) (*keibi.DeleteByQueryResponse, error) {
	if len(ids) == 0 {
		return nil, fmt.Errorf("no ids to delete")
	}

	query := map[string]interface{}{
		"query": map[string]interface{}{
			"terms": map[string][]string{
				"_id": ids,
			},
		},
	}

	res, err := keibi.DeleteByQuery(context.TODO(), client.ES(), []string{index}, query,
		client.ES().DeleteByQuery.WithRefresh(true),
		client.ES().DeleteByQuery.WithConflicts("proceed"),
	)
	if err != nil {
		return nil, err
	}
	return &res, nil
}
