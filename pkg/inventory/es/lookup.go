package es

import (
	"context"
	"encoding/json"
	"github.com/kaytu-io/kaytu-engine/pkg/describe"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type ResourceTypeCountsResponse struct {
	Aggregations struct {
		ResourceTypeGroup struct {
			Buckets []struct {
				Key      string `json:"key"`
				DocCount int    `json:"doc_count"`
			} `json:"buckets"`
		} `json:"resource_type_group"`
	} `json:"aggregations"`
}

func GetResourceTypeCounts(client kaytu.Client, connectors []source.Type, connectionIDs []string, resourceTypes []string, size int) (map[string]int, error) {
	var filters []any
	if len(connectors) > 0 {
		connectorsStr := make([]string, 0, len(connectors))
		for _, connector := range connectors {
			connectorsStr = append(connectorsStr, connector.String())
		}
		filters = append(filters, map[string]any{
			"terms": map[string][]string{
				"source_type": connectorsStr,
			},
		})
	}
	if len(connectionIDs) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{
				"source_id": connectionIDs,
			},
		})
	}
	if len(resourceTypes) > 0 {
		resourceTypes = utils.ToLowerStringSlice(resourceTypes)
		filters = append(filters, map[string]any{
			"terms": map[string][]string{
				"resource_type": resourceTypes,
			},
		})
	}

	query := map[string]any{
		"size": 0,
		"aggs": map[string]any{
			"resource_type_group": map[string]any{
				"terms": map[string]any{
					"field": "resource_type",
					"size":  size,
				},
			},
		},
		"query": map[string]any{
			"bool": map[string][]any{
				"filter": filters,
			},
		},
	}

	queryStr, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}

	var response ResourceTypeCountsResponse
	err = client.Search(context.Background(), describe.InventorySummaryIndex,
		string(queryStr), &response)
	if err != nil {
		return nil, err
	}

	res := make(map[string]int)
	for _, bucket := range response.Aggregations.ResourceTypeGroup.Buckets {
		res[bucket.Key] = bucket.DocCount
	}
	return res, nil
}
