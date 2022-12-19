package es

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
	summarizer "gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"
)

type FetchCostByServicesQueryResponse struct {
	Aggregation struct {
		ServiceGroup struct {
			Buckets []struct {
				Key                string `json:"key"`
				EndTimeAggregation struct {
					Hits struct {
						Total keibi.SearchTotal `json:"total"`
						Hits  []struct {
							ID      string                        `json:"_id"`
							Score   float64                       `json:"_score"`
							Index   string                        `json:"_index"`
							Type    string                        `json:"_type"`
							Version int64                         `json:"_version,omitempty"`
							Source  summarizer.ServiceCostSummary `json:"_source"`
						} `json:"hits"`
					} `json:"hits"`
				} `json:"period_end_max"`
			} `json:"buckets"`
		} `json:"service_grouping"`
	} `json:"aggregations"`
}

func FetchCostByServicesBetween(client keibi.Client, sourceID *string, provider *source.Type, services []string, before time.Time, after time.Time, size int) (map[string]summarizer.ServiceCostSummary, error) {
	hits := make(map[string]summarizer.ServiceCostSummary)
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"report_type": {string(summarizer.CostProviderSummary)}},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"service_name": services},
	})
	filters = append(filters, map[string]interface{}{
		"range": map[string]interface{}{
			"period_end": map[string]string{
				"gte": strconv.FormatInt(after.Unix(), 10),
				"lte": strconv.FormatInt(before.Unix(), 10),
			},
		},
	})

	if sourceID != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_id": {*sourceID}},
		})
	}
	if provider != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_type": {(*provider).String()}},
		})
	}

	res["size"] = size
	res["query"] = map[string]interface{}{
		"bool": map[string]interface{}{
			"filter": filters,
		},
	}
	res["aggs"] = map[string]interface{}{
		"service_grouping": map[string]interface{}{
			"terms": map[string]interface{}{
				"field": "service_name",
			},
			"aggs": map[string]interface{}{
				"period_end_max": map[string]interface{}{
					"top_hits": map[string]interface{}{
						"size": 1,
						"sort": []map[string]interface{}{
							{
								"period_end": map[string]interface{}{
									"order": "desc",
								},
							},
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
	fmt.Println("query=", query)

	var response FetchCostByServicesQueryResponse
	err = client.Search(context.Background(), summarizer.CostSummeryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	for _, bucket := range response.Aggregation.ServiceGroup.Buckets {
		for _, hit := range bucket.EndTimeAggregation.Hits.Hits {
			hits[hit.Source.ServiceName] = hit.Source
		}
	}

	for _, hit := range hits {
		switch strings.ToLower(hit.ResourceType) {
		case "aws::costexplorer::byservicemonthly":
			hitCostStr, err := json.Marshal(hit.Cost)
			if err != nil {
				return nil, err
			}
			var hitCost model.CostExplorerByServiceMonthlyDescription
			err = json.Unmarshal(hitCostStr, &hitCost)
			if err != nil {
				return nil, err
			}
			hit.Cost = hitCost
		}
	}

	return hits, nil
}
