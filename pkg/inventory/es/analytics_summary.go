package es

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/analytics/es/resource"
	inventoryApi "github.com/kaytu-io/kaytu-engine/pkg/inventory/api"
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

func FetchConnectionAnalyticMetricCountAtTime(client kaytu.Client, metricIDs []string,
	connectors []source.Type, connectionIDs, resourceCollections []string, t time.Time, size int) (map[string]int, error) {
	idx := resource.AnalyticsConnectionSummaryIndex
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

	includeConnectionMap := make(map[string]bool)
	for _, connectionID := range connectionIDs {
		includeConnectionMap[connectionID] = true
	}
	includeConnectorMap := make(map[string]bool)
	for _, connector := range connectors {
		includeConnectorMap[connector.String()] = true
	}
	includeResourceCollectionMap := make(map[string]bool)
	for _, resourceCollection := range resourceCollections {
		idx = resource.ResourceCollectionsAnalyticsConnectionSummaryIndex
		includeResourceCollectionMap[resourceCollection] = true
	}

	result := make(map[string]int)

	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	query := string(b)

	fmt.Println("FetchConnectionAnalyticMetricCountAtTime = ", query)
	var response FetchConnectionAnalyticMetricCountAtTimeResponse
	err = client.Search(context.Background(), idx, query, &response)
	if err != nil {
		return nil, err
	}
	for _, metricBucket := range response.Aggregations.MetricGroup.Buckets {
		for _, hit := range metricBucket.Latest.Hits.Hits {
			handleConnResults := func(connResults resource.ConnectionMetricTrendSummaryResult) {
				for _, connectionResults := range connResults.Connections {
					if (len(connectionIDs) > 0 && !includeConnectionMap[connectionResults.ConnectionID]) ||
						(len(connectors) > 0 && !includeConnectorMap[connectionResults.Connector.String()]) {
						continue
					}
					result[hit.Source.MetricID] += connectionResults.ResourceCount
				}
			}

			if len(resourceCollections) > 0 {
				for rcId, rcResult := range hit.Source.ResourceCollections {
					if !includeResourceCollectionMap[rcId] {
						continue
					}
					handleConnResults(rcResult)
				}
			} else if hit.Source.Connections != nil {
				handleConnResults(*hit.Source.Connections)
			} else {
				return nil, errors.New("no connections or resource collections found")
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
							Source resource.ConnectorMetricTrendSummary `json:"_source"`
						} `json:"hits"`
					} `json:"hits"`
				} `json:"latest"`
			} `json:"buckets"`
		} `json:"metric_group"`
	} `json:"aggregations"`
}

func FetchConnectorAnalyticMetricCountAtTime(client kaytu.Client,
	metricIDs []string, connectors []source.Type, resourceCollections []string, t time.Time, size int) (map[string]int, error) {
	idx := resource.AnalyticsConnectorSummaryIndex
	res := make(map[string]any)
	var filters []any

	if len(metricIDs) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"metric_id": metricIDs},
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

	includeConnectorMap := make(map[string]bool)
	for _, connector := range connectors {
		includeConnectorMap[connector.String()] = true
	}
	includeResourceCollectionMap := make(map[string]bool)
	for _, resourceCollection := range resourceCollections {
		idx = resource.ResourceCollectionsAnalyticsConnectorSummaryIndex
		includeResourceCollectionMap[resourceCollection] = true
	}

	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	query := string(b)

	var response FetchConnectorAnalyticMetricCountAtTimeResponse
	err = client.Search(context.Background(), idx, query, &response)
	if err != nil {
		return nil, err
	}

	result := make(map[string]int)
	for _, metricBucket := range response.Aggregations.MetricGroup.Buckets {
		for _, hit := range metricBucket.Latest.Hits.Hits {
			handleConnResults := func(connResults resource.ConnectorMetricTrendSummaryResult) {
				for _, connectorResults := range connResults.Connectors {
					if len(connectors) > 0 && !includeConnectorMap[connectorResults.Connector.String()] {
						continue
					}
					result[hit.Source.MetricID] += connectorResults.ResourceCount
				}
			}

			if len(resourceCollections) > 0 {
				for rcId, rcResult := range hit.Source.ResourceCollections {
					if !includeResourceCollectionMap[rcId] {
						continue
					}
					handleConnResults(rcResult)
				}
			} else if hit.Source.Connectors != nil {
				handleConnResults(*hit.Source.Connectors)
			} else {
				return nil, errors.New("no connectors or resource collections found")
			}
		}
	}
	return result, nil
}

type DatapointWithFailures struct {
	Cost                       float64
	Count                      int
	TotalSuccessfulConnections int64
	TotalConnections           int64

	connectionSuccess map[string]bool
	connectorSuccess  map[string]int64
	connectorTotal    map[string]int64
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

func FetchConnectionMetricTrendSummaryPage(client kaytu.Client, connectionIDs []string, connectors []source.Type, metricIDs, resourceCollections []string, startTime, endTime time.Time, datapointCount, size int) (map[int]DatapointWithFailures, error) {
	idx := resource.AnalyticsConnectionSummaryIndex
	res := make(map[string]any)
	var filters []any

	if len(connectionIDs) == 0 {
		return nil, fmt.Errorf("no connection IDs provided")
	}

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

	includeConnectionMap := make(map[string]bool)
	for _, connectionID := range connectionIDs {
		includeConnectionMap[connectionID] = true
	}
	includeConnectorMap := make(map[string]bool)
	for _, connector := range connectors {
		includeConnectorMap[connector.String()] = true
	}
	includeResourceCollectionMap := make(map[string]bool)
	for _, resourceCollection := range resourceCollections {
		idx = resource.ResourceCollectionsAnalyticsConnectionSummaryIndex
		includeResourceCollectionMap[resourceCollection] = true
	}

	hits := make(map[int]DatapointWithFailures)

	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}
	query := string(b)

	fmt.Println("FetchConnectionMetricTrendSummaryPage = ", query)
	var response ConnectionMetricTrendSummaryQueryResponse
	err = client.Search(context.Background(), idx, query, &response)
	if err != nil {
		return nil, err
	}
	for _, metricBucket := range response.Aggregations.MetricGroup.Buckets {
		for _, evaluatedAtRangeBucket := range metricBucket.EvaluatedAtRangeGroup.Buckets {
			rangeKey := int((evaluatedAtRangeBucket.From + evaluatedAtRangeBucket.To) / 2)
			for _, hit := range evaluatedAtRangeBucket.Latest.Hits.Hits {
				v, ok := hits[rangeKey]
				if !ok {
					v = DatapointWithFailures{
						connectionSuccess: map[string]bool{},
					}
				}

				handleConnResults := func(connResults resource.ConnectionMetricTrendSummaryResult) {
					for _, connectionResults := range connResults.Connections {
						if (len(connectionIDs) > 0 && !includeConnectionMap[connectionResults.ConnectionID]) ||
							(len(connectors) > 0 && !includeConnectorMap[connectionResults.Connector.String()]) {
							continue
						}
						v.Count += connectionResults.ResourceCount
						if _, ok := v.connectionSuccess[connectionResults.ConnectionID]; !ok {
							v.connectionSuccess[connectionResults.ConnectionID] = connectionResults.IsJobSuccessful
						} else {
							v.connectionSuccess[connectionResults.ConnectionID] = v.connectionSuccess[connectionResults.ConnectionID] && connectionResults.IsJobSuccessful
						}
					}
				}

				if len(resourceCollections) > 0 {
					for rcId, rcResult := range hit.Source.ResourceCollections {
						if !includeResourceCollectionMap[rcId] {
							continue
						}
						handleConnResults(rcResult)
					}
				} else if hit.Source.Connections != nil {
					handleConnResults(*hit.Source.Connections)
				} else {
					return nil, errors.New("no connections or resource collections found")
				}
				hits[rangeKey] = v
			}
		}
	}

	for k, v := range hits {
		v.TotalConnections = int64(len(v.connectionSuccess))
		for _, success := range v.connectionSuccess {
			if success {
				v.TotalSuccessfulConnections++
			}
		}
		v.connectionSuccess = nil
		hits[k] = v
	}

	return hits, nil
}

type ConnectorMetricTrendSummaryQueryResponse struct {
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
									Source resource.ConnectorMetricTrendSummary `json:"_source"`
								} `json:"hits"`
							} `json:"hits"`
						} `json:"latest"`
					} `json:"buckets"`
				} `json:"evaluated_at_range_group"`
			} `json:"buckets"`
		} `json:"metric_group"`
	} `json:"aggregations"`
}

func FetchConnectorMetricTrendSummaryPage(client kaytu.Client, connectors []source.Type, metricIDs []string, resourceCollections []string, startTime time.Time, endTime time.Time, datapointCount int, size int) (map[int]DatapointWithFailures, error) {
	idx := resource.AnalyticsConnectorSummaryIndex
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

	includeConnectorMap := make(map[string]bool)
	for _, connector := range connectors {
		includeConnectorMap[connector.String()] = true
	}
	includeResourceCollectionMap := make(map[string]bool)
	for _, resourceCollection := range resourceCollections {
		idx = resource.ResourceCollectionsAnalyticsConnectorSummaryIndex
		includeResourceCollectionMap[resourceCollection] = true
	}

	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	query := string(b)

	var response ConnectorMetricTrendSummaryQueryResponse
	err = client.Search(context.Background(), idx, query, &response)
	if err != nil {
		return nil, err
	}

	hits := make(map[int]DatapointWithFailures)
	for _, metricBucket := range response.Aggregations.MetricGroup.Buckets {
		for _, evaluatedAtRangeBucket := range metricBucket.EvaluatedAtRangeGroup.Buckets {
			rangeKey := int((evaluatedAtRangeBucket.From + evaluatedAtRangeBucket.To) / 2)
			for _, hit := range evaluatedAtRangeBucket.Latest.Hits.Hits {
				v, ok := hits[rangeKey]
				if !ok {
					v = DatapointWithFailures{
						connectorTotal:   map[string]int64{},
						connectorSuccess: map[string]int64{},
					}
					hits[rangeKey] = v
				}

				handleConnResults := func(connResults resource.ConnectorMetricTrendSummaryResult) {
					for _, connectorResults := range connResults.Connectors {
						if len(connectors) > 0 && !includeConnectorMap[connectorResults.Connector.String()] {
							continue
						}
						v.Count += connectorResults.ResourceCount
						v.connectorTotal[connectorResults.Connector.String()] = max(v.connectorTotal[connectorResults.Connector.String()], connectorResults.TotalConnections)
						if _, ok := v.connectorSuccess[connectorResults.Connector.String()]; !ok {
							v.connectorSuccess[connectorResults.Connector.String()] = connectorResults.TotalSuccessfulConnections
						} else {
							v.connectorSuccess[connectorResults.Connector.String()] = min(v.connectorSuccess[connectorResults.Connector.String()], connectorResults.TotalSuccessfulConnections)
						}
					}
				}

				if len(resourceCollections) > 0 {
					for rcId, rcResult := range hit.Source.ResourceCollections {
						if !includeResourceCollectionMap[rcId] {
							continue
						}
						handleConnResults(rcResult)
					}
				} else if hit.Source.Connectors != nil {
					handleConnResults(*hit.Source.Connectors)
				} else {
					return nil, errors.New("no connectors or resource collections found")
				}
				hits[rangeKey] = v
			}
		}
	}

	fmt.Println(hits)
	for k, v := range hits {
		for _, total := range v.connectorTotal {
			v.TotalConnections += total
		}
		for _, total := range v.connectorSuccess {
			v.TotalSuccessfulConnections += total
		}
		v.connectorSuccess = nil
		v.connectorTotal = nil
		hits[k] = v
	}
	return hits, nil
}

type FetchConnectionAnalyticsResourcesCountAtTimeResponse struct {
	Took         int `json:"took"`
	Aggregations struct {
		MetricIDGroup struct {
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
		} `json:"metric_id_group"`
	} `json:"aggregations"`
}

type FetchConnectionAnalyticsResourcesCountAtTimeReturnValue struct {
	ResourceCountsSum int
	LatestEvaluatedAt int64
}

func FetchConnectionAnalyticsResourcesCountAtTime(client kaytu.Client, connectors []source.Type, connectionIDs []string,
	resourceCollections []string, metricIDs []string, t time.Time, size int) (map[string]FetchConnectionAnalyticsResourcesCountAtTimeReturnValue, error) {
	idx := resource.AnalyticsConnectionSummaryIndex
	if len(resourceCollections) > 0 {
		idx = resource.ResourceCollectionsAnalyticsConnectionSummaryIndex
	}

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

	includeConnectionMap := make(map[string]bool)
	for _, connectionID := range connectionIDs {
		includeConnectionMap[connectionID] = true
	}
	includeConnectorMap := make(map[string]bool)
	for _, connector := range connectors {
		includeConnectorMap[connector.String()] = true
	}
	includeResourceCollectionMap := make(map[string]bool)
	for _, resourceCollection := range resourceCollections {
		includeResourceCollectionMap[resourceCollection] = true
	}

	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	query := string(b)
	fmt.Println("FetchConnectionAnalyticsResourcesCountAtTime query =", query)
	var response FetchConnectionAnalyticsResourcesCountAtTimeResponse
	err = client.Search(
		context.Background(),
		idx,
		query,
		&response)
	if err != nil {
		return nil, err
	}

	result := make(map[string]FetchConnectionAnalyticsResourcesCountAtTimeReturnValue)
	for _, metricBucket := range response.Aggregations.MetricIDGroup.Buckets {
		for _, hit := range metricBucket.Latest.Hits.Hits {
			for _, connectionResults := range hit.Source.Connections.Connections {
				if (len(connectionIDs) > 0 && !includeConnectionMap[connectionResults.ConnectionID]) ||
					(len(connectors) > 0 && !includeConnectorMap[connectionResults.Connector.String()]) {
					continue
				}
				v := result[connectionResults.ConnectionID]
				v.ResourceCountsSum += connectionResults.ResourceCount
				v.LatestEvaluatedAt = max(v.LatestEvaluatedAt, hit.Source.EvaluatedAt)
				result[connectionResults.ConnectionID] = v
			}
			for rcId, rcResult := range hit.Source.ResourceCollections {
				if !includeResourceCollectionMap[rcId] {
					continue
				}
				for _, connectionResults := range rcResult.Connections {
					if (len(connectionIDs) > 0 && !includeConnectionMap[connectionResults.ConnectionID]) ||
						(len(connectors) > 0 && !includeConnectorMap[connectionResults.Connector.String()]) {
						continue
					}
					v := result[connectionResults.ConnectionID]
					v.ResourceCountsSum += connectionResults.ResourceCount
					v.LatestEvaluatedAt = max(v.LatestEvaluatedAt, hit.Source.EvaluatedAt)
					result[connectionResults.ConnectionID] = v
				}
			}
		}
	}

	return result, nil
}

type AssetTableByDimensionQueryResponse struct {
	Aggregations struct {
		MetricIdGroup struct {
			Buckets []struct {
				Key       string `json:"key"`
				DateGroup struct {
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
				} `json:"date_group"`
			} `json:"buckets"`
		} `json:"metric_id_group"`
	} `json:"aggregations"`
}

func FetchAssetTableByDimension(client kaytu.Client, metricIds []string, granularity inventoryApi.TableGranularityType, dimension inventoryApi.DimensionType, startTime, endTime time.Time) ([]DimensionTrend, error) {
	query := make(map[string]any)
	var filters []any

	index := ""
	switch dimension {
	case inventoryApi.DimensionTypeConnection:
		index = resource.AnalyticsConnectionSummaryIndex
	case inventoryApi.DimensionTypeMetric:
		index = resource.AnalyticsConnectorSummaryIndex
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
	if granularity == inventoryApi.TableGranularityTypeMonthly {
		dateGroupField = "month"
	} else if granularity == inventoryApi.TableGranularityTypeYearly {
		dateGroupField = "year"
	}

	query["size"] = 0
	query["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}
	query["aggs"] = map[string]any{
		"metric_id_group": map[string]any{
			"terms": map[string]any{
				"field": "metric_id",
				"size":  EsFetchPageSize,
			},
			"aggs": map[string]any{
				"date_group": map[string]any{
					"terms": map[string]any{
						"field": dateGroupField,
						"size":  EsFetchPageSize,
					},
					"aggs": map[string]any{
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

	resultMap := make(map[string]DimensionTrend)
	for _, bucket := range response.Aggregations.MetricIdGroup.Buckets {
		for _, dateBucket := range bucket.DateGroup.Buckets {
			for _, hit := range dateBucket.Latest.Hits.Hits {
				switch dimension {
				case inventoryApi.DimensionTypeConnection:
					for _, connectionResults := range hit.Source.Connections.Connections {
						mt, ok := resultMap[connectionResults.ConnectionID]
						if !ok {
							mt = DimensionTrend{
								DimensionID:   connectionResults.ConnectionID,
								DimensionName: connectionResults.ConnectionName,
								Trend:         make(map[string]float64),
							}
						}
						mt.Trend[dateBucket.Key] += float64(connectionResults.ResourceCount)
						resultMap[connectionResults.ConnectionID] = mt
					}
				case inventoryApi.DimensionTypeMetric:
					mt, ok := resultMap[hit.Source.MetricID]
					if !ok {
						mt = DimensionTrend{
							DimensionID:   hit.Source.MetricID,
							DimensionName: hit.Source.MetricName,
							Trend:         make(map[string]float64),
						}
					}
					mt.Trend[dateBucket.Key] += float64(hit.Source.Connections.TotalResourceCount)
					resultMap[hit.Source.MetricID] = mt
				}

			}
		}
	}
	result := make([]DimensionTrend, 0, len(resultMap))
	for _, v := range resultMap {
		result = append(result, v)
	}

	return result, nil
}
