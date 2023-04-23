package es

import (
	"context"
	"encoding/json"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
)

type ResourceAggregationResponse struct {
	Aggregations ResourceAggregations `json:"aggregations"`
}
type ResourceAggregations struct {
	ResourceIDFilter AggregationResult `json:"resource_id_filter"`
}
type AggregationResult struct {
	DocCountErrorUpperBound int      `json:"doc_count_error_upper_bound"`
	SumOtherDocCount        int      `json:"sum_other_doc_count"`
	Buckets                 []Bucket `json:"buckets"`
}
type Bucket struct {
	Key      string `json:"key"`
	DocCount int    `json:"doc_count"`
}

func GetResourceIDsForAccountResourceTypeFromES(client keibi.Client, sourceID, resourceType string) (*ResourceAggregationResponse, error) {
	terms := map[string][]string{
		"source_id":     {sourceID},
		"resource_type": {resourceType},
	}

	root := map[string]interface{}{}
	root["size"] = 0

	resourceIDFilter := map[string]interface{}{
		"terms": map[string]interface{}{"field": "resource_id", "size": 1000000},
	}
	aggs := map[string]interface{}{
		"resource_id_filter": resourceIDFilter,
	}
	root["aggs"] = aggs

	boolQuery := make(map[string]interface{})
	var filters []map[string]interface{}
	for k, vs := range terms {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{
				k: vs,
			},
		})
	}
	boolQuery["filter"] = filters
	root["query"] = map[string]interface{}{
		"bool": boolQuery,
	}

	queryBytes, err := json.Marshal(root)
	if err != nil {
		return nil, err
	}

	var response ResourceAggregationResponse
	err = client.Search(context.Background(), InventorySummaryIndex,
		string(queryBytes), &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}
