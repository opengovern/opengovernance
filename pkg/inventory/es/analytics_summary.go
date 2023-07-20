package es

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/analytics/es"
	summarizer "github.com/kaytu-io/kaytu-engine/pkg/summarizer/es"
	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"strconv"
	"time"
)

type FetchConnectionAnalyticMetricCountAtTimeResponse struct {
	Aggregations struct {
		MetricGroup struct {
			Buckets []struct {
				Key    string `json:"key"`
				Latest struct {
					Hits struct {
						Hits []struct {
							Source es.ConnectionMetricTrendSummary `json:"_source"`
						} `json:"hits"`
					} `json:"hits"`
				} `json:"latest"`
			} `json:"buckets"`
		} `json:"metric_group"`
	} `json:"aggregations"`
}

func FetchConnectionAnalyticMetricCountAtTime(client keibi.Client, connectors []source.Type, connectionIDs []string, t time.Time, metricNames []string, size int) (map[string]int, error) {
	res := make(map[string]any)
	var filters []any

	if len(connectionIDs) == 0 {
		return nil, fmt.Errorf("no connection IDs provided")
	}

	filters = append(filters, map[string]any{
		"terms": map[string][]string{"report_type": {string(summarizer.MetricTrendConnectionSummary)}},
	})
	if len(metricNames) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"metric_name": metricNames},
		})
	}

	if len(connectors) > 0 {
		connectorStrings := make([]string, 0, len(connectors))
		for _, provider := range connectors {
			connectorStrings = append(connectorStrings, provider.String())
		}
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"connector": connectorStrings},
		})
	}
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"evaluated_at": map[string]string{
				"lte": strconv.FormatInt(t.UnixMilli(), 10),
			},
		},
	})
	res["size"] = 0
	res["aggs"] = map[string]any{
		"metric_group": map[string]any{
			"terms": map[string]any{
				"field": "metric_name",
				"size":  size,
			},
			"aggs": map[string]any{
				"latest": map[string]any{
					"top_hits": map[string]any{
						"size": 1,
						"sort": map[string]string{
							"evaluated_at": "desc",
						},
					},
				},
			},
		},
	}

	result := make(map[string]int)
	for _, connectionId := range connectionIDs {
		localFilter := append(filters, map[string]any{
			"term": map[string]string{"connection_id": connectionId},
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

		var response FetchConnectionAnalyticMetricCountAtTimeResponse
		err = client.Search(context.Background(), summarizer.ConnectionSummaryIndex, query, &response)
		if err != nil {
			return nil, err
		}
		for _, metricBucket := range response.Aggregations.MetricGroup.Buckets {
			for _, hit := range metricBucket.Latest.Hits.Hits {
				result[hit.Source.MetricName] += hit.Source.ResourceCount
			}
		}
	}

	return result, nil
}

type FetchConnectorAnalyticMetricCountAtTimeResponse struct {
	Aggregations struct {
		MetricGroup struct {
			Buckets []struct {
				Key    string `json:"key"`
				Latest struct {
					Hits struct {
						Hits []struct {
							Source es.ConnectorMetricTrendSummary `json:"_source"`
						} `json:"hits"`
					} `json:"hits"`
				} `json:"latest"`
			} `json:"buckets"`
		} `json:"metric_group"`
	} `json:"aggregations"`
}

func FetchConnectorAnalyticMetricCountAtTime(client keibi.Client, connectors []source.Type, t time.Time, metricNames []string, size int) (map[string]int, error) {
	res := make(map[string]any)
	var filters []any

	filters = append(filters, map[string]any{
		"terms": map[string][]string{"report_type": {string(summarizer.MetricTrendConnectorSummary)}},
	})
	if len(metricNames) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"metric_name": metricNames},
		})
	}
	if len(connectors) > 0 {
		connectorStrings := make([]string, 0, len(connectors))
		for _, provider := range connectors {
			connectorStrings = append(connectorStrings, provider.String())
		}
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"connector": connectorStrings},
		})
	}
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"evaluated_at": map[string]string{
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
		"metric_group": map[string]any{
			"terms": map[string]any{
				"field": "metric_name",
				"size":  size,
			},
			"aggs": map[string]any{
				"latest": map[string]any{
					"top_hits": map[string]any{
						"size": 1,
						"sort": map[string]string{
							"evaluated_at": "desc",
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

	var response FetchConnectorAnalyticMetricCountAtTimeResponse
	err = client.Search(context.Background(), summarizer.ProviderSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	result := make(map[string]int)
	for _, metricBucket := range response.Aggregations.MetricGroup.Buckets {
		for _, hit := range metricBucket.Latest.Hits.Hits {
			result[hit.Source.MetricName] += hit.Source.ResourceCount
		}
	}
	return result, nil
}
