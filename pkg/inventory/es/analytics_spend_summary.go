package es

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/analytics/es/spend"
	"github.com/kaytu-io/kaytu-engine/pkg/summarizer/es"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"strconv"
	"time"
)

type ConnectionDailySpendHistoryByMetric struct {
	ConnectionID  string
	Connector     string
	MetricID      string
	MetricName    string
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
							Hits struct {
								Hits []struct {
									Source spend.ConnectionMetricTrendSummary `json:"_source"`
								} `json:"hits"`
							} `json:"hits"`
						} `json:"start_cost_group"`
						EndCostGroup struct {
							Hits struct {
								Hits []struct {
									Source spend.ConnectionMetricTrendSummary `json:"_source"`
								} `json:"hits"`
							} `json:"hits"`
						} `json:"end_cost_group"`
					} `json:"buckets"`
				} `json:"metric_id_group"`
			} `json:"buckets"`
		} `json:"connection_id_group"`
	} `json:"aggregations"`
}

func FetchConnectionDailySpendHistoryByMetric(client kaytu.Client, connectionIDs []string, connectors []source.Type, metricIDs []string, startTime time.Time, endTime time.Time, size int) ([]ConnectionDailySpendHistoryByMetric, error) {
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

			for _, v := range metricBucket.StartCostGroup.Hits.Hits {
				if startTime.Format("2006-01-02") == v.Source.Date {
					hit.StartDateCost = v.Source.CostValue
				}
				hit.Connector = v.Source.Connector.String()
				hit.MetricName = v.Source.MetricName
			}
			for _, v := range metricBucket.EndCostGroup.Hits.Hits {
				if endTime.Format("2006-01-02") == v.Source.Date {
					hit.EndDateCost = v.Source.CostValue
				}
				hit.MetricName = v.Source.MetricName
			}
			hits = append(hits, hit)
		}
	}

	return hits, nil
}

type ConnectionDailySpendHistory struct {
	ConnectionID  string
	Connector     string
	TotalCost     float64
	StartDateCost float64
	EndDateCost   float64
}

type FetchConnectionDailySpendHistoryQueryResponse struct {
	Aggregations struct {
		ConnectionIDGroup struct {
			Buckets []struct {
				Key               string `json:"key"`
				CostValueSumGroup struct {
					Value float64 `json:"value"`
				} `json:"cost_value_sum_group"`
				StartCostGroup struct {
					Hits struct {
						Hits []struct {
							Source spend.ConnectionMetricTrendSummary `json:"_source"`
						} `json:"hits"`
					} `json:"hits"`
				} `json:"start_cost_group"`
				EndCostGroup struct {
					Hits struct {
						Hits []struct {
							Source spend.ConnectionMetricTrendSummary `json:"_source"`
						} `json:"hits"`
					} `json:"hits"`
				} `json:"end_cost_group"`
			} `json:"buckets"`
		} `json:"connection_id_group"`
	} `json:"aggregations"`
}

func FetchConnectionDailySpendHistory(client kaytu.Client, connectionIDs []string, connectors []source.Type, metricIDs []string, startTime time.Time, endTime time.Time, size int) ([]ConnectionDailySpendHistory, error) {
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
	}

	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	query := string(b)
	fmt.Println("FetchConnectionDailySpendHistory =", query)
	var response FetchConnectionDailySpendHistoryQueryResponse
	err = client.Search(context.Background(), spend.AnalyticsSpendConnectionSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	var hits []ConnectionDailySpendHistory
	for _, connectionBucket := range response.Aggregations.ConnectionIDGroup.Buckets {
		hit := ConnectionDailySpendHistory{
			ConnectionID:  connectionBucket.Key,
			Connector:     "",
			TotalCost:     connectionBucket.CostValueSumGroup.Value,
			StartDateCost: 0,
			EndDateCost:   0,
		}

		for _, v := range connectionBucket.StartCostGroup.Hits.Hits {
			if startTime.Format("2006-01-02") == v.Source.Date {
				hit.StartDateCost = v.Source.CostValue
			}
			hit.Connector = v.Source.Connector.String()
		}
		for _, v := range connectionBucket.EndCostGroup.Hits.Hits {
			if endTime.Format("2006-01-02") == v.Source.Date {
				hit.EndDateCost = v.Source.CostValue
			}
		}
		hits = append(hits, hit)
	}

	return hits, nil
}

type ConnectorDailySpendHistoryByMetric struct {
	Connector     string
	MetricID      string
	MetricName    string
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
							Hits struct {
								Hits []struct {
									Source spend.ConnectorMetricTrendSummary `json:"_source"`
								} `json:"hits"`
							} `json:"hits"`
						} `json:"start_cost_group"`
						EndCostGroup struct {
							Hits struct {
								Hits []struct {
									Source spend.ConnectorMetricTrendSummary `json:"_source"`
								} `json:"hits"`
							} `json:"hits"`
						} `json:"end_cost_group"`
					} `json:"buckets"`
				} `json:"metric_id_group"`
			} `json:"buckets"`
		} `json:"connector_group"`
	} `json:"aggregations"`
}

func FetchConnectorDailySpendHistoryByMetric(client kaytu.Client, connectors []source.Type, metricIDs []string, startTime time.Time, endTime time.Time, size int) ([]ConnectorDailySpendHistoryByMetric, error) {
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

			for _, v := range metricBucket.StartCostGroup.Hits.Hits {
				hit.StartDateCost = v.Source.CostValue
				hit.MetricName = v.Source.MetricName
			}
			for _, v := range metricBucket.EndCostGroup.Hits.Hits {
				hit.EndDateCost = v.Source.CostValue
				hit.MetricName = v.Source.MetricName
			}
			hits = append(hits, hit)
		}
	}

	return hits, nil
}

type ConnectionSpendTrendQueryResponse struct {
	Aggregations struct {
		DateGroup struct {
			Buckets []struct {
				Key          string `json:"key"`
				CostSumGroup struct {
					Value float64 `json:"value"`
				} `json:"cost_sum_group"`
			} `json:"buckets"`
		} `json:"date_group"`
	} `json:"aggregations"`
}

func FetchConnectionSpendTrend(client kaytu.Client, metricIds []string, connectionIDs []string, connectors []source.Type, startTime, endTime time.Time) (map[string]float64, error) {
	query := make(map[string]any)
	var filters []any

	if len(connectionIDs) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"connection_id": connectionIDs},
		})
	}
	if len(metricIds) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"metric_id": metricIds},
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

	query["size"] = 0
	query["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}
	query["aggs"] = map[string]any{
		"date_group": map[string]any{
			"terms": map[string]any{
				"field": "date",
				"size":  es.EsFetchPageSize,
			},
			"aggs": map[string]any{
				"cost_sum_group": map[string]any{
					"sum": map[string]string{
						"field": "cost_value",
					},
				},
			},
		},
	}

	queryJson, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}
	fmt.Printf("FetchConnectionSpendTrend = %s\n", queryJson)

	var response ConnectionSpendTrendQueryResponse
	err = client.Search(context.Background(), spend.AnalyticsSpendConnectionSummaryIndex, string(queryJson), &response)
	if err != nil {
		return nil, err
	}

	result := make(map[string]float64)
	for _, bucket := range response.Aggregations.DateGroup.Buckets {
		result[bucket.Key] = bucket.CostSumGroup.Value
	}

	return result, nil
}

type ConnectorSpendTrendQueryResponse struct {
	Aggregations struct {
		DateGroup struct {
			Buckets []struct {
				Key          string `json:"key"`
				CostSumGroup struct {
					Value float64 `json:"value"`
				} `json:"cost_sum_group"`
			} `json:"buckets"`
		} `json:"date_group"`
	} `json:"aggregations"`
}

func FetchConnectorSpendTrend(client kaytu.Client, metricIds []string, connectors []source.Type, startTime, endTime time.Time) (map[string]float64, error) {
	query := make(map[string]any)
	var filters []any

	if len(connectors) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]source.Type{"connector": connectors},
		})
	}
	if len(metricIds) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"metric_id": metricIds},
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

	query["size"] = 0
	query["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}
	query["aggs"] = map[string]any{
		"date_group": map[string]any{
			"terms": map[string]any{
				"field": "date",
				"size":  es.EsFetchPageSize,
			},
			"aggs": map[string]any{
				"cost_sum_group": map[string]any{
					"sum": map[string]string{
						"field": "cost_value",
					},
				},
			},
		},
	}

	queryJson, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}
	fmt.Printf("FetchConnectorSpendTrend = %s\n", queryJson)

	var response ConnectorSpendTrendQueryResponse
	err = client.Search(context.Background(), spend.AnalyticsSpendConnectorSummaryIndex, string(queryJson), &response)
	if err != nil {
		return nil, err
	}

	result := make(map[string]float64)
	for _, bucket := range response.Aggregations.DateGroup.Buckets {
		result[bucket.Key] = bucket.CostSumGroup.Value
	}

	return result, nil
}

type FetchSpendByMetricQueryResponse struct {
	Aggregations struct {
		MetricIDGroup struct {
			Buckets []struct {
				Key               string `json:"key"`
				CostValueSumGroup struct {
					Value float64 `json:"value"`
				} `json:"cost_value_sum_group"`
			} `json:"buckets"`
		} `json:"metric_id_group"`
	} `json:"aggregations"`
}

func FetchSpendByMetric(client kaytu.Client, connectionIDs []string, connectors []source.Type, metricIDs []string, startTime time.Time, endTime time.Time, size int) (map[string]float64, error) {
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
				"top_doc": map[string]any{
					"top_hits": map[string]any{
						"size": size,
						"sort": map[string]any{
							"period_start": "asc",
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
	fmt.Println("FetchSpendByMetric =", query)
	var response FetchSpendByMetricQueryResponse
	err = client.Search(context.Background(), spend.AnalyticsSpendConnectionSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	resp := map[string]float64{}
	for _, metricBucket := range response.Aggregations.MetricIDGroup.Buckets {
		resp[metricBucket.Key] = metricBucket.CostValueSumGroup.Value
	}
	return resp, nil
}
