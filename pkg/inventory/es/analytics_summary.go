package es

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/analytics/es/resource"
	"github.com/kaytu-io/kaytu-engine/pkg/analytics/es/spend"
	inventoryApi "github.com/kaytu-io/kaytu-engine/pkg/inventory/api"
	"github.com/kaytu-io/kaytu-engine/pkg/summarizer/es"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"math"
	"strconv"
	"time"

	"github.com/kaytu-io/kaytu-util/pkg/source"
)

const timeAtMaxSearchFrame = 5 * 24 * time.Hour // 5 days

type FetchConnectionAnalyticMetricCountAtTimeResponse struct {
	Aggregations struct {
		MetricGroup struct {
			Buckets []struct {
				Key    string `json:"key"`
				Latest struct {
					Hits struct {
						Hits []struct {
							Source resource.ConnectionMetricTrendSummary `json:"_source"`
						} `json:"hits"`
					} `json:"hits"`
				} `json:"latest"`
			} `json:"buckets"`
		} `json:"metric_group"`
	} `json:"aggregations"`
}

func FetchConnectionAnalyticMetricCountAtTime(client kaytu.Client, connectors []source.Type, connectionIDs []string, t time.Time, metricIDs []string, size int) (map[string]int, error) {
	res := make(map[string]any)
	var filters []any

	if len(connectionIDs) == 0 {
		return nil, fmt.Errorf("no connection IDs provided")
	}

	if len(metricIDs) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"metric_id": metricIDs},
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
				"gte": strconv.FormatInt(t.Add(-1*timeAtMaxSearchFrame).UnixMilli(), 10),
			},
		},
	})
	res["size"] = 0
	res["aggs"] = map[string]any{
		"metric_group": map[string]any{
			"terms": map[string]any{
				"field": "metric_id",
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

		fmt.Println("FetchConnectionAnalyticMetricCountAtTime = ", query)
		var response FetchConnectionAnalyticMetricCountAtTimeResponse
		err = client.Search(context.Background(), resource.AnalyticsConnectionSummaryIndex, query, &response)
		if err != nil {
			return nil, err
		}
		for _, metricBucket := range response.Aggregations.MetricGroup.Buckets {
			for _, hit := range metricBucket.Latest.Hits.Hits {
				result[hit.Source.MetricID] += hit.Source.ResourceCount
			}
		}
	}

	return result, nil
}

type FetchConnectorAnalyticMetricCountAtTimeResponse struct {
	Aggregations struct {
		MetricGroup struct {
			Buckets []struct {
				Key            string `json:"key"`
				ConnectorGroup struct {
					Buckets []struct {
						Key    string `json:"key"`
						Latest struct {
							Hits struct {
								Hits []struct {
									Source resource.ConnectorMetricTrendSummary `json:"_source"`
								} `json:"hits"`
							} `json:"hits"`
						} `json:"latest"`
					} `json:"buckets"`
				} `json:"connector_group"`
			} `json:"buckets"`
		} `json:"metric_group"`
	} `json:"aggregations"`
}

func FetchConnectorAnalyticMetricCountAtTime(client kaytu.Client, connectors []source.Type, t time.Time, metricIDs []string, size int) (map[string]int, error) {
	res := make(map[string]any)
	var filters []any

	if len(metricIDs) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"metric_id": metricIDs},
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
				"gte": strconv.FormatInt(t.Add(-1*timeAtMaxSearchFrame).UnixMilli(), 10),
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
				"field": "metric_id",
				"size":  size,
			},
			"aggs": map[string]any{
				"connector_group": map[string]any{
					"terms": map[string]any{
						"field": "connector",
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
			},
		},
	}

	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	query := string(b)

	var response FetchConnectorAnalyticMetricCountAtTimeResponse
	err = client.Search(context.Background(), resource.AnalyticsConnectorSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	result := make(map[string]int)
	for _, metricBucket := range response.Aggregations.MetricGroup.Buckets {
		for _, connector := range metricBucket.ConnectorGroup.Buckets {
			for _, hit := range connector.Latest.Hits.Hits {
				result[hit.Source.MetricID] += hit.Source.ResourceCount
			}
		}
	}
	return result, nil
}

type ConnectionMetricTrendSummaryQueryResponse struct {
	Aggregations struct {
		MetricGroup struct {
			Buckets []struct {
				Key                   string `json:"key"`
				EvaluatedAtRangeGroup struct {
					Buckets []struct {
						From   float64 `json:"from"`
						To     float64 `json:"to"`
						Latest struct {
							Hits struct {
								Hits []struct {
									Source resource.ConnectionMetricTrendSummary `json:"_source"`
								} `json:"hits"`
							} `json:"hits"`
						} `json:"latest"`
					} `json:"buckets"`
				} `json:"evaluated_at_range_group"`
			} `json:"buckets"`
		} `json:"metric_group"`
	} `json:"aggregations"`
}

func FetchConnectionMetricTrendSummaryPage(client kaytu.Client, connectionIDs, metricIDs []string, startTime, endTime time.Time, datapointCount int, size int) (map[int]int, error) {
	res := make(map[string]any)
	var filters []any

	filters = append(filters, map[string]any{
		"terms": map[string][]string{"metric_id": metricIDs},
	})
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"evaluated_at": map[string]string{
				"gte": strconv.FormatInt(startTime.UnixMilli(), 10),
				"lte": strconv.FormatInt(endTime.UnixMilli(), 10),
			},
		},
	})
	res["size"] = 0
	startTimeUnixMilli := startTime.UnixMilli()
	endTimeUnixMilli := endTime.UnixMilli()
	step := int(math.Ceil(float64(endTimeUnixMilli-startTimeUnixMilli) / float64(datapointCount)))
	ranges := make([]map[string]any, 0, datapointCount)
	for i := 0; i < datapointCount; i++ {
		ranges = append(ranges, map[string]any{
			"from": float64(startTimeUnixMilli + int64(step*i)),
			"to":   float64(startTimeUnixMilli + int64(step*(i+1))),
		})
	}
	res["aggs"] = map[string]any{
		"metric_group": map[string]any{
			"terms": map[string]any{
				"field": "metric_id",
				"size":  size,
			},
			"aggs": map[string]any{
				"evaluated_at_range_group": map[string]any{
					"range": map[string]any{
						"field":  "evaluated_at",
						"ranges": ranges,
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
			},
		},
	}

	hits := make(map[int]int)
	for _, connectionID := range connectionIDs {
		localFilters := append(filters, map[string]any{
			"term": map[string]string{"connection_id": connectionID},
		})
		res["query"] = map[string]any{
			"bool": map[string]any{
				"filter": localFilters,
			},
		}

		b, err := json.Marshal(res)
		if err != nil {
			return nil, err
		}
		query := string(b)

		fmt.Println("FetchConnectionMetricTrendSummaryPage = ", query)
		var response ConnectionMetricTrendSummaryQueryResponse
		err = client.Search(context.Background(), resource.AnalyticsConnectionSummaryIndex, query, &response)
		if err != nil {
			return nil, err
		}
		for _, metricBucket := range response.Aggregations.MetricGroup.Buckets {
			for _, evaluatedAtRangeBucket := range metricBucket.EvaluatedAtRangeGroup.Buckets {
				rangeKey := int((evaluatedAtRangeBucket.From + evaluatedAtRangeBucket.To) / 2)
				for _, hit := range evaluatedAtRangeBucket.Latest.Hits.Hits {
					hits[rangeKey] += hit.Source.ResourceCount
				}
			}
		}
	}

	return hits, nil
}

type ConnectorMetricTrendSummaryQueryResponse struct {
	Aggregations struct {
		MetricGroup struct {
			Buckets []struct {
				Key            string `json:"key"`
				ConnectorGroup struct {
					Buckets []struct {
						Key                   string `json:"key"`
						EvaluatedAtRangeGroup struct {
							Buckets []struct {
								From   float64 `json:"from"`
								To     float64 `json:"to"`
								Latest struct {
									Hits struct {
										Hits []struct {
											Source resource.ConnectorMetricTrendSummary `json:"_source"`
										} `json:"hits"`
									} `json:"hits"`
								} `json:"latest"`
							} `json:"buckets"`
						} `json:"evaluated_at_range_group"`
					} `json:"buckets"`
				} `json:"connector_group"`
			} `json:"buckets"`
		} `json:"metric_group"`
	} `json:"aggregations"`
}

func FetchConnectorMetricTrendSummaryPage(client kaytu.Client, connectors []source.Type, metricIDs []string, startTime, endTime time.Time, datapointCount int, size int) (map[int]int, error) {
	res := make(map[string]any)
	var filters []any

	filters = append(filters, map[string]any{
		"terms": map[string][]string{"metric_id": metricIDs},
	})

	if len(connectors) > 0 {
		connectorsStr := make([]string, 0, len(connectors))
		for _, connector := range connectors {
			connectorsStr = append(connectorsStr, string(connector))
		}
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"connector": connectorsStr},
		})
	}
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"evaluated_at": map[string]string{
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

	startTimeUnixMilli := startTime.UnixMilli()
	endTimeUnixMilli := endTime.UnixMilli()
	step := int(math.Ceil(float64(endTimeUnixMilli-startTimeUnixMilli) / float64(datapointCount)))
	ranges := make([]map[string]any, 0, datapointCount)
	for i := 0; i < datapointCount; i++ {
		ranges = append(ranges, map[string]any{
			"from": float64(startTimeUnixMilli + int64(step*i)),
			"to":   float64(startTimeUnixMilli + int64(step*(i+1))),
		})
	}
	res["aggs"] = map[string]any{
		"metric_group": map[string]any{
			"terms": map[string]any{
				"field": "metric_id",
				"size":  size,
			},
			"aggs": map[string]any{
				"connector_group": map[string]any{
					"terms": map[string]any{
						"field": "connector",
						"size":  size,
					},
					"aggs": map[string]any{
						"evaluated_at_range_group": map[string]any{
							"range": map[string]any{
								"field":  "evaluated_at",
								"ranges": ranges,
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

	var response ConnectorMetricTrendSummaryQueryResponse
	err = client.Search(context.Background(), resource.AnalyticsConnectorSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	hits := make(map[int]int)
	for _, metricBucket := range response.Aggregations.MetricGroup.Buckets {
		for _, connector := range metricBucket.ConnectorGroup.Buckets {
			for _, evaluatedAtRangeBucket := range connector.EvaluatedAtRangeGroup.Buckets {
				rangeKey := int((evaluatedAtRangeBucket.From + evaluatedAtRangeBucket.To) / 2)
				for _, hit := range evaluatedAtRangeBucket.Latest.Hits.Hits {
					hits[rangeKey] += hit.Source.ResourceCount
				}
			}
		}
	}

	return hits, nil
}

type RegionSummaryQueryResponse struct {
	Aggregations struct {
		MetricGroup struct {
			Buckets []struct {
				Key               string `json:"key"`
				ConnectionIDGroup struct {
					Buckets []struct {
						Key         string `json:"key"`
						RegionGroup struct {
							Buckets []struct {
								Key    string `json:"key"`
								Latest struct {
									Hits struct {
										Hits []struct {
											Source resource.RegionMetricTrendSummary `json:"_source"`
										} `json:"hits"`
									} `json:"hits"`
								} `json:"latest"`
							} `json:"buckets"`
						} `json:"region_group"`
					} `json:"buckets"`
				} `json:"connection_id_group"`
			} `json:"buckets"`
		} `json:"metric_group"`
	} `json:"aggregations"`
}

func FetchRegionSummaryPage(client kaytu.Client, connectors []source.Type, connectionIDs []string, sort []map[string]any, timeAt time.Time, size int) (map[string]int, error) {
	res := make(map[string]any)

	var filters []any

	if len(connectors) > 0 {
		connectorStr := make([]string, 0, len(connectors))
		for _, connector := range connectors {
			connectorStr = append(connectorStr, connector.String())
		}
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"connector": connectorStr},
		})
	}
	if len(connectionIDs) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"connection_id": connectionIDs},
		})
	}
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"evaluated_at": map[string]any{
				"lte": strconv.FormatInt(timeAt.UnixMilli(), 10),
				"gte": strconv.FormatInt(timeAt.Add(-1*timeAtMaxSearchFrame).UnixMilli(), 10),
			},
		},
	})
	res["size"] = 0
	if sort != nil {
		res["sort"] = sort
	}
	res["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}
	res["aggs"] = map[string]any{
		"metric_group": map[string]any{
			"terms": map[string]any{
				"field": "metric_id",
				"size":  size,
			},
			"aggs": map[string]any{
				"connection_id_group": map[string]any{
					"terms": map[string]any{
						"field": "connection_id",
						"size":  size,
					},
					"aggs": map[string]any{
						"region_group": map[string]any{
							"terms": map[string]any{
								"field": "region",
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

	fmt.Println("FetchRegionSummaryPage query = ", query)
	var response RegionSummaryQueryResponse
	err = client.Search(context.Background(), resource.AnalyticsRegionSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	hits := make(map[string]int)
	for _, metricBucket := range response.Aggregations.MetricGroup.Buckets {
		for _, connectionIDBucket := range metricBucket.ConnectionIDGroup.Buckets {
			for _, regionBucket := range connectionIDBucket.RegionGroup.Buckets {
				for _, hit := range regionBucket.Latest.Hits.Hits {
					hits[hit.Source.Region] += hit.Source.ResourceCount
				}
			}
		}
	}
	return hits, nil
}

type FetchConnectionAnalyticsResourcesCountAtTimeResponse struct {
	Took         int `json:"took"`
	Aggregations struct {
		ConnectionIDGroup struct {
			Buckets []struct {
				Key             string `json:"key"`
				SumMetricGroups struct {
					Value float64 `json:"value"`
				} `json:"sum_metric_groups"`
				LatestAvailable struct {
					Value float64 `json:"value"`
				} `json:"latest_available"`
			} `json:"buckets"`
		} `json:"connection_id_group"`
	} `json:"aggregations"`
}

type FetchConnectionAnalyticsResourcesCountAtTimeReturnValue struct {
	ResourceCountsSum int
	LatestEvaluatedAt int64
}

func FetchConnectionAnalyticsResourcesCountAtTime(client kaytu.Client, connectors []source.Type, connectionIDs []string, metricIDs []string, t time.Time, size int) (map[string]FetchConnectionAnalyticsResourcesCountAtTimeReturnValue, error) {
	res := make(map[string]any)
	var filters []any
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"evaluated_at": map[string]any{
				"lte": t.UnixMilli(),
				"gte": t.Add(-1 * timeAtMaxSearchFrame).UnixMilli(),
			},
		},
	})

	if len(metricIDs) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"metric_id": metricIDs},
		})
	}

	if len(connectors) > 0 {
		connectorsStr := make([]string, 0, len(connectors))
		for _, c := range connectors {
			connectorsStr = append(connectorsStr, c.String())
		}
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"connector": connectorsStr},
		})
	}

	if len(connectionIDs) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"connection_id": connectionIDs},
		})
	}

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
				"metric_group": map[string]any{
					"terms": map[string]any{
						"field": "metric_id",
						"size":  size,
					},
					"aggs": map[string]any{
						"latest_quantity": map[string]any{
							"scripted_metric": map[string]any{
								"init_script":    "state.quantities = new TreeMap()",
								"map_script":     "state.quantities.put(doc.evaluated_at.value, [doc.evaluated_at.value, doc.resource_count.value])",
								"combine_script": "return state.quantities.lastEntry().getValue()",
								"reduce_script":  "long maxkey = 0; long qty = 0; for (a in states) {def currentKey = a[0]; if (currentKey > maxkey) {maxkey = currentKey; qty = a[1]} } return qty;",
							},
						},
					},
				},
				"sum_metric_groups": map[string]any{
					"sum_bucket": map[string]any{
						"buckets_path": "metric_group>latest_quantity.value",
					},
				},
				"latest_available": map[string]any{
					"max": map[string]any{
						"field": "evaluated_at",
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
	fmt.Println("FetchConnectionAnalyticsResourcesCountAtTime query =", query)
	var response FetchConnectionAnalyticsResourcesCountAtTimeResponse
	err = client.SearchWithFilterPath(
		context.Background(),
		resource.AnalyticsConnectionSummaryIndex,
		query,
		[]string{
			"took",
			"aggregations.connection_id_group.buckets.key",
			"aggregations.connection_id_group.buckets.latest_available.value",
			"aggregations.connection_id_group.buckets.sum_metric_groups.value",
		},
		&response)
	if err != nil {
		return nil, err
	}

	hits := make(map[string]FetchConnectionAnalyticsResourcesCountAtTimeReturnValue)
	for _, connectionIdBucket := range response.Aggregations.ConnectionIDGroup.Buckets {
		hits[connectionIdBucket.Key] = FetchConnectionAnalyticsResourcesCountAtTimeReturnValue{
			ResourceCountsSum: int(connectionIdBucket.SumMetricGroups.Value),
			LatestEvaluatedAt: int64(connectionIdBucket.LatestAvailable.Value),
		}
	}
	return hits, nil
}

type FetchConnectorAnalyticsResourcesCountAtResponse struct {
	Aggregations struct {
		ConnectorGroup struct {
			Key     string `json:"key"`
			Buckets []struct {
				Key         string `json:"key"`
				MetricGroup struct {
					Key     string `json:"key"`
					Buckets []struct {
						Latest struct {
							Hits struct {
								Hits []struct {
									Source resource.ConnectorMetricTrendSummary `json:"_source"`
								} `json:"hits"`
							} `json:"hits"`
						} `json:"latest"`
					} `json:"buckets"`
				} `json:"metric_group"`
			} `json:"buckets"`
		} `json:"connector_group"`
	} `json:"aggregations"`
}

func FetchConnectorAnalyticsResourcesCountAtTime(client kaytu.Client, connectors []source.Type, t time.Time, size int) ([]resource.ConnectorMetricTrendSummary, error) {
	var hits []resource.ConnectorMetricTrendSummary
	res := make(map[string]any)
	var filters []any

	filters = append(filters, map[string]any{
		"range": map[string]any{
			"evaluated_at": map[string]any{
				"lte": t.UnixMilli(),
				"gte": t.Add(-1 * timeAtMaxSearchFrame).UnixMilli(),
			},
		},
	})

	if len(connectors) > 0 {
		connectorsStr := make([]string, 0, len(connectors))
		for _, c := range connectors {
			connectorsStr = append(connectorsStr, c.String())
		}
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"connector": connectorsStr},
		})
	}

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
				"metric_group": map[string]any{
					"terms": map[string]any{
						"field": "metric_id",
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
			},
		},
	}

	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	query := string(b)
	fmt.Println("FetchConnectorAnalyticsResourcesCountAtResponse query =", query)
	var response FetchConnectorAnalyticsResourcesCountAtResponse
	err = client.Search(context.Background(), resource.AnalyticsConnectorSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	for _, connectorBucket := range response.Aggregations.ConnectorGroup.Buckets {
		for _, metricBucket := range connectorBucket.MetricGroup.Buckets {
			for _, hit := range metricBucket.Latest.Hits.Hits {
				hits = append(hits, hit.Source)
			}
		}
	}
	return hits, nil
}

type AssetTableByDimensionQueryResponse struct {
	Aggregations struct {
		DimensionGroup struct {
			Buckets []struct {
				Key       string `json:"key"`
				DateGroup struct {
					Buckets []struct {
						Key      string `json:"key"`
						SumGroup struct {
							Value float64 `json:"value"`
						} `json:"sum_group"`
						Latest struct {
							Hits struct {
								Hits []struct {
									Source spend.ConnectionMetricTrendSummary `json:"_source"`
								} `json:"hits"`
							} `json:"hits"`
						} `json:"latest"`
					} `json:"buckets"`
				} `json:"date_group"`
			} `json:"buckets"`
		} `json:"dimension_group"`
	} `json:"aggregations"`
}

func FetchAssetTableByDimension(client kaytu.Client, metricIds []string, granularity inventoryApi.SpendTableGranularity, dimension inventoryApi.SpendDimension, startTime, endTime time.Time) ([]DimensionTrend, error) {
	query := make(map[string]any)
	var filters []any

	dimensionField := ""
	index := ""
	switch dimension {
	case inventoryApi.SpendDimensionConnection:
		dimensionField = "connection_id"
		index = spend.AnalyticsSpendConnectionSummaryIndex
	case inventoryApi.SpendDimensionMetric:
		dimensionField = "metric_id"
		index = spend.AnalyticsSpendConnectorSummaryIndex
	default:
		return nil, errors.New("dimension is not supported")
	}
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"evaluated_at": map[string]string{
				"gte": strconv.FormatInt(startTime.UnixMilli(), 10),
				"lte": strconv.FormatInt(endTime.UnixMilli(), 10),
			},
		},
	})
	if len(metricIds) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string]any{
				"metric_id": metricIds,
			},
		})
	}

	dateGroupField := "date"
	if granularity == inventoryApi.SpendTableGranularityMonthly {
		dateGroupField = "month"
	} else if granularity == inventoryApi.SpendTableGranularityYearly {
		dateGroupField = "year"
	}

	query["size"] = 0
	query["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}
	query["aggs"] = map[string]any{
		"dimension_group": map[string]any{
			"terms": map[string]any{
				"field": dimensionField,
				"size":  es.EsFetchPageSize,
			},
			"aggs": map[string]any{
				"date_group": map[string]any{
					"terms": map[string]any{
						"field": dateGroupField,
						"size":  es.EsFetchPageSize,
					},
					"aggs": map[string]any{
						"sum_group": map[string]any{
							"sum": map[string]string{
								"field": "resource_count",
							},
						},
						"latest": map[string]any{
							"top_hits": map[string]any{
								"size": 1,
								"sort": map[string]string{
									"_id": "asc",
								},
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
	fmt.Printf("FetchAssetTableByDimension = %s\n", queryJson)

	var response AssetTableByDimensionQueryResponse
	err = client.Search(context.Background(), index, string(queryJson), &response)
	if err != nil {
		return nil, err
	}

	var result []DimensionTrend
	for _, bucket := range response.Aggregations.DimensionGroup.Buckets {
		mt := DimensionTrend{
			DimensionID: bucket.Key,
			Trend:       make(map[string]float64),
		}
		for _, dateBucket := range bucket.DateGroup.Buckets {
			mt.Trend[dateBucket.Key] = dateBucket.SumGroup.Value
			for _, hit := range dateBucket.Latest.Hits.Hits {
				switch dimension {
				case inventoryApi.SpendDimensionConnection:
					mt.DimensionName = hit.Source.ConnectionName
				case inventoryApi.SpendDimensionMetric:
					mt.DimensionName = hit.Source.MetricName
				default:
					return nil, errors.New("dimension is not supported")
				}
			}
		}
		result = append(result, mt)
	}

	return result, nil
}
