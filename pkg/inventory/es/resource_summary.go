package es

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	summarizer "github.com/kaytu-io/kaytu-engine/pkg/summarizer/es"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type FetchConnectionResourceTypeCountAtTimeResponse struct {
	Aggregations struct {
		ResourceTypeGroup struct {
			Buckets []struct {
				Key    string `json:"key"`
				Latest struct {
					Hits struct {
						Hits []struct {
							Source summarizer.ConnectionResourceTypeTrendSummary `json:"_source"`
						} `json:"hits"`
					} `json:"hits"`
				} `json:"latest"`
			} `json:"buckets"`
		} `json:"resource_type_group"`
	} `json:"aggregations"`
}

func FetchConnectionResourceTypeCountAtTime(client kaytu.Client, connectors []source.Type, connectionIDs []string, t time.Time, resourceTypes []string, size int) (map[string]int, error) {
	res := make(map[string]any)
	var filters []any

	if len(connectionIDs) == 0 {
		return nil, fmt.Errorf("no connection IDs provided")
	}

	filters = append(filters, map[string]any{
		"terms": map[string][]string{"report_type": {string(summarizer.ResourceTypeTrendConnectionSummary)}},
	})
	if len(resourceTypes) > 0 {
		resourceTypes = utils.ToLowerStringSlice(resourceTypes)
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"resource_type": resourceTypes},
		})
	}

	if len(connectors) > 0 {
		connectorStrings := make([]string, 0, len(connectors))
		for _, provider := range connectors {
			connectorStrings = append(connectorStrings, provider.String())
		}
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"source_type": connectorStrings},
		})
	}
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"described_at": map[string]string{
				"lte": strconv.FormatInt(t.UnixMilli(), 10),
			},
		},
	})
	res["size"] = 0
	res["aggs"] = map[string]any{
		"resource_type_group": map[string]any{
			"terms": map[string]any{
				"field": "resource_type",
				"size":  size,
			},
			"aggs": map[string]any{
				"latest": map[string]any{
					"top_hits": map[string]any{
						"size": 1,
						"sort": map[string]string{
							"described_at": "desc",
						},
					},
				},
			},
		},
	}

	result := make(map[string]int)
	for _, connectionId := range connectionIDs {
		localFilter := append(filters, map[string]any{
			"term": map[string]string{"source_id": connectionId},
		})
		res["query"] = map[string]any{
			"bool": map[string]any{
				"filter": localFilter,
			},
		}
		b, err := json.Marshal(res)
		if err != nil {
			return nil, err
		}

		query := string(b)
		fmt.Println("query=", query, "index=", summarizer.ConnectionSummaryIndex)
		var response FetchConnectionResourceTypeCountAtTimeResponse
		err = client.Search(context.Background(), summarizer.ConnectionSummaryIndex, query, &response)
		if err != nil {
			return nil, err
		}
		for _, resourceTypeBucket := range response.Aggregations.ResourceTypeGroup.Buckets {
			for _, hit := range resourceTypeBucket.Latest.Hits.Hits {
				result[hit.Source.ResourceType] += hit.Source.ResourceCount
			}
		}
	}

	return result, nil
}

type FetchConnectorResourceTypeCountAtTimeResponse struct {
	Aggregations struct {
		ResourceTypeGroup struct {
			Buckets []struct {
				Key    string `json:"key"`
				Latest struct {
					Hits struct {
						Hits []struct {
							Source summarizer.ProviderResourceTypeTrendSummary `json:"_source"`
						} `json:"hits"`
					} `json:"hits"`
				} `json:"latest"`
			} `json:"buckets"`
		} `json:"resource_type_group"`
	} `json:"aggregations"`
}

func FetchConnectorResourceTypeCountAtTime(client kaytu.Client, connectors []source.Type, t time.Time, resourceTypes []string, size int) (map[string]int, error) {
	res := make(map[string]any)
	var filters []any

	filters = append(filters, map[string]any{
		"terms": map[string][]string{"report_type": {string(summarizer.ResourceTypeTrendProviderSummary)}},
	})
	if len(resourceTypes) > 0 {
		resourceTypes = utils.ToLowerStringSlice(resourceTypes)
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"resource_type": resourceTypes},
		})
	}
	if len(connectors) > 0 {
		connectorStrings := make([]string, 0, len(connectors))
		for _, provider := range connectors {
			connectorStrings = append(connectorStrings, provider.String())
		}
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"source_type": connectorStrings},
		})
	}
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"described_at": map[string]string{
				"lte": strconv.FormatInt(t.UnixMilli(), 10),
			},
		},
	})
	res["size"] = 0
	res["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}
	res["aggs"] = map[string]any{
		"resource_type_group": map[string]any{
			"terms": map[string]any{
				"field": "resource_type",
				"size":  size,
			},
			"aggs": map[string]any{
				"latest": map[string]any{
					"top_hits": map[string]any{
						"size": 1,
						"sort": map[string]string{
							"described_at": "desc",
						},
					},
				},
			},
		},
	}

	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	query := string(b)
	fmt.Println("query=", query, "index=", summarizer.ProviderSummaryIndex)
	var response FetchConnectorResourceTypeCountAtTimeResponse
	err = client.Search(context.Background(), summarizer.ProviderSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	result := make(map[string]int)
	for _, resourceTypeBucket := range response.Aggregations.ResourceTypeGroup.Buckets {
		for _, hit := range resourceTypeBucket.Latest.Hits.Hits {
			result[hit.Source.ResourceType] += hit.Source.ResourceCount
		}
	}
	return result, nil
}
