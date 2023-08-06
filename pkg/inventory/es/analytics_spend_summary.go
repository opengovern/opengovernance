package es

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/analytics/es/spend"
	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"strconv"
	"time"
)

type ConnectionDailySpendHistoryByMetric struct {
	ConnectionID  string
	Connector     string
	MetricID      string
	TotalCost     float64
	StartDateCost float64
	EndDateCost   float64
}

type FetchConnectionDailySpendHistoryByMetricQueryResponse struct {
	Aggregations struct {
		ConnectionIDGroup struct {
			Buckets []struct {
				Key           string `json:"key"`
				MetricIDGroup struct {
					Buckets []struct {
						Key               string `json:"key"`
						CostValueSumGroup struct {
							Value float64 `json:"value"`
						} `json:"cost_value_sum_group"`
						StartCostGroup struct {
							Hits []struct {
								Source spend.ConnectionMetricTrendSummary `json:"_source"`
							} `json:"hits"`
						} `json:"start_cost_group"`
						EndCostGroup struct {
							Hits []struct {
								Source spend.ConnectionMetricTrendSummary `json:"_source"`
							} `json:"hits"`
						} `json:"end_cost_group"`
					} `json:"buckets"`
				} `json:"metric_id_group"`
			} `json:"buckets"`
		} `json:"connection_id_group"`
	} `json:"aggregations"`
}

func FetchConnectionDailySpendHistoryByMetric(client keibi.Client, connectionIDs []string, connectors []source.Type, metricIDs []string, startTime time.Time, endTime time.Time, size int) ([]ConnectionDailySpendHistoryByMetric, error) {
	res := make(map[string]any)
	var filters []any

	if len(metricIDs) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"metric_id": metricIDs},
		})
	}
	if len(connectionIDs) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"connection_id": connectionIDs},
		})
	}
	if len(connectors) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]source.Type{"connector": connectors},
		})
	}
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"period_end": map[string]string{
				"lte": strconv.FormatInt(endTime.UnixMilli(), 10),
			},
		},
	})
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"period_start": map[string]string{
				"gte": strconv.FormatInt(startTime.UnixMilli(), 10),
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
		"connection_id_group": map[string]any{
			"terms": map[string]any{
				"field": "connection_id",
				"size":  size,
			},
			"aggs": map[string]any{
				"metric_id_group": map[string]any{
					"terms": map[string]any{
						"field": "metric_id",
						"size":  size,
					},
					"aggs": map[string]any{
						"cost_value_sum_group": map[string]any{
							"sum": map[string]any{
								"field": "cost_value",
							},
						},
						"start_cost_group": map[string]any{
							"top_hits": map[string]any{
								"size": size,
								"sort": map[string]any{
									"period_start": "asc",
								},
							},
						},
						"end_cost_group": map[string]any{
							"top_hits": map[string]any{
								"size": size,
								"sort": map[string]any{
									"period_end": "desc",
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
	fmt.Println("FetchConnectionDailySpendHistoryByMetric =", query)
	var response FetchConnectionDailySpendHistoryByMetricQueryResponse
	err = client.Search(context.Background(), spend.AnalyticsSpendConnectionSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	var hits []ConnectionDailySpendHistoryByMetric
	for _, connectionBucket := range response.Aggregations.ConnectionIDGroup.Buckets {
		for _, metricBucket := range connectionBucket.MetricIDGroup.Buckets {
			hit := ConnectionDailySpendHistoryByMetric{
				ConnectionID:  connectionBucket.Key,
				Connector:     "",
				MetricID:      metricBucket.Key,
				TotalCost:     metricBucket.CostValueSumGroup.Value,
				StartDateCost: 0,
				EndDateCost:   0,
			}

			for _, v := range metricBucket.StartCostGroup.Hits {
				hit.StartDateCost = v.Source.CostValue
				hit.Connector = v.Source.Connector.String()
			}
			for _, v := range metricBucket.EndCostGroup.Hits {
				hit.EndDateCost = v.Source.CostValue
			}
			hits = append(hits, hit)
		}
	}

	return hits, nil
}

type ConnectorDailySpendHistoryByMetric struct {
	Connector     string
	MetricID      string
	TotalCost     float64
	StartDateCost float64
	EndDateCost   float64
}

type FetchConnectorDailySpendHistoryByMetricQueryResponse struct {
	Aggregations struct {
		ConnectorGroup struct {
			Buckets []struct {
				Key           string `json:"key"`
				MetricIDGroup struct {
					Buckets []struct {
						Key               string `json:"key"`
						CostValueSumGroup struct {
							Value float64 `json:"value"`
						} `json:"cost_value_sum_group"`
						StartCostGroup struct {
							Hits []struct {
								Source spend.ConnectorMetricTrendSummary `json:"_source"`
							} `json:"hits"`
						} `json:"start_cost_group"`
						EndCostGroup struct {
							Hits []struct {
								Source spend.ConnectorMetricTrendSummary `json:"_source"`
							} `json:"hits"`
						} `json:"end_cost_group"`
					} `json:"buckets"`
				} `json:"metric_id_group"`
			} `json:"buckets"`
		} `json:"connector_group"`
	} `json:"aggregations"`
}

func FetchConnectorDailySpendHistoryByMetric(client keibi.Client, connectors []source.Type, metricIDs []string, startTime time.Time, endTime time.Time, size int) ([]ConnectorDailySpendHistoryByMetric, error) {
	res := make(map[string]any)
	var filters []any

	if len(metricIDs) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"metric_id": metricIDs},
		})
	}
	if len(connectors) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]source.Type{"connector": connectors},
		})
	}
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"period_end": map[string]string{
				"lte": strconv.FormatInt(endTime.UnixMilli(), 10),
			},
		},
	})
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"period_start": map[string]string{
				"gte": strconv.FormatInt(startTime.UnixMilli(), 10),
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
		"connector_group": map[string]any{
			"terms": map[string]any{
				"field": "connector",
				"size":  size,
			},
			"aggs": map[string]any{
				"metric_id_group": map[string]any{
					"terms": map[string]any{
						"field": "metric_id",
						"size":  size,
					},
					"aggs": map[string]any{
						"cost_value_sum_group": map[string]any{
							"sum": map[string]any{
								"field": "cost_value",
							},
						},
						"start_cost_group": map[string]any{
							"top_hits": map[string]any{
								"size": size,
								"sort": map[string]any{
									"period_start": "asc",
								},
							},
						},
						"end_cost_group": map[string]any{
							"top_hits": map[string]any{
								"size": size,
								"sort": map[string]any{
									"period_end": "desc",
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
	fmt.Println("FetchConnectorDailySpendHistoryByMetric =", query)
	var response FetchConnectorDailySpendHistoryByMetricQueryResponse
	err = client.Search(context.Background(), spend.AnalyticsSpendConnectionSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	var hits []ConnectorDailySpendHistoryByMetric
	for _, connectorBucket := range response.Aggregations.ConnectorGroup.Buckets {
		for _, metricBucket := range connectorBucket.MetricIDGroup.Buckets {
			hit := ConnectorDailySpendHistoryByMetric{
				Connector:     connectorBucket.Key,
				MetricID:      metricBucket.Key,
				TotalCost:     metricBucket.CostValueSumGroup.Value,
				StartDateCost: 0,
				EndDateCost:   0,
			}

			for _, v := range metricBucket.StartCostGroup.Hits {
				hit.StartDateCost = v.Source.CostValue
			}
			for _, v := range metricBucket.EndCostGroup.Hits {
				hit.EndDateCost = v.Source.CostValue
			}
			hits = append(hits, hit)
		}
	}

	return hits, nil
}
