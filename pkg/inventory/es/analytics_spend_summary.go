package es

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/analytics/es/spend"
	inventoryApi "github.com/kaytu-io/kaytu-engine/pkg/inventory/api"
	"github.com/kaytu-io/kaytu-engine/pkg/summarizer/es"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"strconv"
	"time"
)

type ConnectionDailySpendHistoryByMetric struct {
	ConnectionID  string
	Connector     source.Type
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
							HitSelect struct {
								Hits struct {
									Hits []struct {
										Source spend.ConnectionMetricTrendSummary `json:"_source"`
									} `json:"hits"`
								} `json:"hits"`
							} `json:"hit_select"`
						} `json:"start_cost_group"`
						EndCostGroup struct {
							HitSelect struct {
								Hits struct {
									Hits []struct {
										Source spend.ConnectionMetricTrendSummary `json:"_source"`
									} `json:"hits"`
								} `json:"hits"`
							} `json:"hit_select"`
						} `json:"end_cost_group"`
						AvailableConnectorsGroup struct {
							Buckets []struct {
								Key string `json:"key"`
							} `json:"buckets"`
						} `json:"available_connectors_group"`
						MetricNameGroup struct {
							Buckets []struct {
								Key string `json:"key"`
							} `json:"buckets"`
						} `json:"metric_name_group"`
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
			"date_epoch": map[string]string{
				"gte": strconv.FormatInt(startTime.UnixMilli(), 10),
				"lte": strconv.FormatInt(endTime.UnixMilli(), 10),
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
							"filter": map[string]any{
								"term": map[string]string{
									"date": startTime.Format("2006-01-02"),
								},
							},
							"aggs": map[string]any{
								"hit_select": map[string]any{
									"top_hits": map[string]any{
										"size": 1,
									},
								},
							},
						},
						"end_cost_group": map[string]any{
							"filter": map[string]any{
								"term": map[string]string{
									"date": endTime.Format("2006-01-02"),
								},
							},
							"aggs": map[string]any{
								"hit_select": map[string]any{
									"top_hits": map[string]any{
										"size": 1,
									},
								},
							},
						},
						"available_connectors_group": map[string]any{
							"terms": map[string]any{
								"field": "connector",
								"size":  1,
							},
						},
						"metric_name_group": map[string]any{
							"terms": map[string]any{
								"field": "metric_name",
								"size":  1,
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
				Connector:     source.Nil,
				MetricID:      metricBucket.Key,
				MetricName:    "",
				TotalCost:     metricBucket.CostValueSumGroup.Value,
				StartDateCost: 0,
				EndDateCost:   0,
			}
			for _, v := range metricBucket.StartCostGroup.HitSelect.Hits.Hits {
				hit.StartDateCost = v.Source.CostValue
				hit.MetricName = v.Source.MetricName
			}
			for _, v := range metricBucket.EndCostGroup.HitSelect.Hits.Hits {
				hit.EndDateCost = v.Source.CostValue
				hit.MetricName = v.Source.MetricName
			}
			if len(metricBucket.MetricNameGroup.Buckets) > 0 {
				hit.MetricName = metricBucket.MetricNameGroup.Buckets[0].Key
			}
			for _, v := range metricBucket.AvailableConnectorsGroup.Buckets {
				if hit.Connector == source.Nil {
					c, _ := source.ParseType(v.Key)
					hit.Connector = c
					break
				}
			}
			hits = append(hits, hit)
		}
	}

	return hits, nil
}

type ConnectionDailySpendHistory struct {
	ConnectionID  string
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
					CostValueSumGroup struct {
						Value float64 `json:"value"`
					} `json:"cost_value_sum_group"`
				} `json:"start_cost_group"`
				EndCostGroup struct {
					CostValueSumGroup struct {
						Value float64 `json:"value"`
					} `json:"cost_value_sum_group"`
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
			"date_epoch": map[string]string{
				"gte": strconv.FormatInt(startTime.UnixMilli(), 10),
				"lte": strconv.FormatInt(endTime.UnixMilli(), 10),
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
					"filter": map[string]any{
						"term": map[string]string{
							"date": startTime.Format("2006-01-02"),
						},
					},
					"aggs": map[string]any{
						"cost_value_sum_group": map[string]any{
							"sum": map[string]any{
								"field": "cost_value",
							},
						},
					},
				},
				"end_cost_group": map[string]any{
					"filter": map[string]any{
						"term": map[string]string{
							"date": endTime.Format("2006-01-02"),
						},
					},
					"aggs": map[string]any{
						"cost_value_sum_group": map[string]any{
							"sum": map[string]any{
								"field": "cost_value",
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
			TotalCost:     connectionBucket.CostValueSumGroup.Value,
			StartDateCost: connectionBucket.StartCostGroup.CostValueSumGroup.Value,
			EndDateCost:   connectionBucket.EndCostGroup.CostValueSumGroup.Value,
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
						Key             string `json:"key"`
						MetricNameGroup struct {
							Buckets []struct {
								Key string `json:"key"`
							} `json:"buckets"`
						} `json:"metric_name_group"`
						CostValueSumGroup struct {
							Value float64 `json:"value"`
						} `json:"cost_value_sum_group"`
						StartCostGroup struct {
							CostValueSumGroup struct {
								Value float64 `json:"value"`
							} `json:"cost_value_sum_group"`
						} `json:"start_cost_group"`
						EndCostGroup struct {
							CostValueSumGroup struct {
								Value float64 `json:"value"`
							} `json:"cost_value_sum_group"`
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
			"date_epoch": map[string]string{
				"gte": strconv.FormatInt(startTime.UnixMilli(), 10),
				"lte": strconv.FormatInt(endTime.UnixMilli(), 10),
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
							"filter": map[string]any{
								"term": map[string]string{
									"date": startTime.Format("2006-01-02"),
								},
							},
							"aggs": map[string]any{
								"cost_value_sum_group": map[string]any{
									"sum": map[string]any{
										"field": "cost_value",
									},
								},
							},
						},
						"end_cost_group": map[string]any{
							"filter": map[string]any{
								"term": map[string]string{
									"date": endTime.Format("2006-01-02"),
								},
							},
							"aggs": map[string]any{
								"cost_value_sum_group": map[string]any{
									"sum": map[string]any{
										"field": "cost_value",
									},
								},
							},
						},
						"metric_name_group": map[string]any{
							"terms": map[string]any{
								"field": "metric_name",
								"size":  1,
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

			hit.StartDateCost = metricBucket.StartCostGroup.CostValueSumGroup.Value
			hit.EndDateCost = metricBucket.EndCostGroup.CostValueSumGroup.Value
			if len(metricBucket.MetricNameGroup.Buckets) > 0 {
				hit.MetricName = metricBucket.MetricNameGroup.Buckets[0].Key
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
				JobSuccessful struct {
					Buckets []struct {
						Key             string `json:"key_as_string"`
						ConnectionCount struct {
							Value int64 `json:"value"`
						} `json:"connection_count"`
					} `json:"buckets"`
				} `json:"job_successful"`
			} `json:"buckets"`
		} `json:"date_group"`
	} `json:"aggregations"`
}

func FetchConnectionSpendTrend(client kaytu.Client, granularity inventoryApi.SpendTableGranularity, metricIds []string, connectionIDs []string, connectors []source.Type, startTime, endTime time.Time) (map[string]DatapointWithFailures, error) {
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
			"date_epoch": map[string]string{
				"gte": strconv.FormatInt(startTime.UnixMilli(), 10),
				"lte": strconv.FormatInt(endTime.UnixMilli(), 10),
			},
		},
	})

	granularityField := "date"
	if granularity == inventoryApi.SpendTableGranularityMonthly {
		granularityField = "month"
	} else if granularity == inventoryApi.SpendTableGranularityYearly {
		granularityField = "year"
	}

	query["size"] = 0
	query["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}
	query["aggs"] = map[string]any{
		"date_group": map[string]any{
			"terms": map[string]any{
				"field": granularityField,
				"size":  es.EsFetchPageSize,
			},
			"aggs": map[string]any{
				"cost_sum_group": map[string]any{
					"sum": map[string]string{
						"field": "cost_value",
					},
				},
				"job_successful": map[string]any{
					"terms": map[string]any{
						"field": "is_job_successful",
						"size":  es.EsFetchPageSize,
					},
					"aggs": map[string]any{
						"connection_count": map[string]any{
							"cardinality": map[string]any{
								"field":               "connection_id",
								"precision_threshold": 100,
								"rehash":              false,
							},
						},
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

	result := make(map[string]DatapointWithFailures)
	for _, bucket := range response.Aggregations.DateGroup.Buckets {
		var totalConnections, totalSuccessfulConnections int64
		for _, jobSuccess := range bucket.JobSuccessful.Buckets {
			if jobSuccess.Key == "true" {
				totalSuccessfulConnections += jobSuccess.ConnectionCount.Value
			}
			totalConnections += jobSuccess.ConnectionCount.Value
		}
		result[bucket.Key] = DatapointWithFailures{
			Cost:                       bucket.CostSumGroup.Value,
			TotalSuccessfulConnections: totalSuccessfulConnections,
			TotalConnections:           totalConnections,
		}
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
				Jobs struct {
					Key     string `json:"key"`
					Buckets []struct {
						MaxConnections struct {
							Value int64 `json:"value"`
						} `json:"max_total"`
						MaxSuccessfulConnections struct {
							Value int64 `json:"value"`
						} `json:"max_successful"`
					} `json:"buckets"`
				} `json:"jobs"`
			} `json:"buckets"`
		} `json:"date_group"`
	} `json:"aggregations"`
}

func FetchConnectorSpendTrend(client kaytu.Client, granularity inventoryApi.SpendTableGranularity, metricIds []string, connectors []source.Type, startTime, endTime time.Time) (map[string]DatapointWithFailures, error) {
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
			"date_epoch": map[string]string{
				"gte": strconv.FormatInt(startTime.UnixMilli(), 10),
				"lte": strconv.FormatInt(endTime.UnixMilli(), 10),
			},
		},
	})

	granularityField := "date"
	if granularity == inventoryApi.SpendTableGranularityMonthly {
		granularityField = "month"
	} else if granularity == inventoryApi.SpendTableGranularityYearly {
		granularityField = "year"
	}

	query["size"] = 0
	query["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}
	query["aggs"] = map[string]any{
		"date_group": map[string]any{
			"terms": map[string]any{
				"field": granularityField,
				"size":  es.EsFetchPageSize,
			},
			"aggs": map[string]any{
				"cost_sum_group": map[string]any{
					"sum": map[string]string{
						"field": "cost_value",
					},
				},
				"jobs": map[string]any{
					"terms": map[string]any{
						"field": "connector",
						"size":  es.EsFetchPageSize,
					},
					"aggs": map[string]any{
						"max_total": map[string]any{
							"max": map[string]string{
								"field": "total_connections",
							},
						},
						"max_successful": map[string]any{
							"max": map[string]string{
								"field": "total_successful_connections",
							},
						},
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

	result := make(map[string]DatapointWithFailures)
	for _, bucket := range response.Aggregations.DateGroup.Buckets {
		var totalConnections, totalSuccessfulConnections int64
		for _, job := range bucket.Jobs.Buckets {
			totalConnections += job.MaxConnections.Value
			totalSuccessfulConnections += job.MaxSuccessfulConnections.Value
		}
		result[bucket.Key] = DatapointWithFailures{
			Cost:                       bucket.CostSumGroup.Value,
			TotalSuccessfulConnections: totalSuccessfulConnections,
			TotalConnections:           totalConnections,
		}
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
				TopDoc struct {
					Top []struct {
						Sort    []int64 `json:"sort"`
						Metrics struct {
							MetricName string `json:"metric_name"`
						} `json:"metrics"`
					} `json:"top"`
				} `json:"top_doc"`
			} `json:"buckets"`
		} `json:"metric_id_group"`
	} `json:"aggregations"`
}

type SpendMetricResp struct {
	MetricName string
	CostValue  float64
}

func FetchSpendByMetric(client kaytu.Client, connectionIDs []string, connectors []source.Type, metricIDs []string, startTime time.Time, endTime time.Time, size int) (map[string]SpendMetricResp, error) {
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
			"date_epoch": map[string]string{
				"gte": strconv.FormatInt(startTime.UnixMilli(), 10),
				"lte": strconv.FormatInt(endTime.UnixMilli(), 10),
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
					"top_metrics": map[string]any{
						"size": 1,
						"metrics": []map[string]any{
							{"field": "metric_name"},
						},
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

	resp := map[string]SpendMetricResp{}
	for _, metricBucket := range response.Aggregations.MetricIDGroup.Buckets {
		metricName := metricBucket.Key
		for _, item := range metricBucket.TopDoc.Top {
			metricName = item.Metrics.MetricName
		}
		resp[metricBucket.Key] = SpendMetricResp{
			MetricName: metricName,
			CostValue:  metricBucket.CostValueSumGroup.Value,
		}
	}
	return resp, nil
}
