package es

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/opengovern/og-util/pkg/integration"
	"github.com/opengovern/og-util/pkg/opengovernance-es-sdk"
	"math"
	"strconv"
	"time"

	"github.com/opengovern/opengovernance/pkg/analytics/es/resource"
	inventoryApi "github.com/opengovern/opengovernance/pkg/inventory/api"
	"go.uber.org/zap"
)

const timeAtMaxSearchFrame = 5 * 24 * time.Hour // 5 days

type CountWithTime struct {
	Count int
	Time  time.Time
}

type FetchConnectionAnalyticMetricCountAtTimeResponse struct {
	Aggregations struct {
		MetricGroup struct {
			Buckets []struct {
				Key    string `json:"key"`
				Latest struct {
					Hits struct {
						Hits []struct {
							Source resource.IntegrationMetricTrendSummary `json:"_source"`
						} `json:"hits"`
					} `json:"hits"`
				} `json:"latest"`
			} `json:"buckets"`
		} `json:"metric_group"`
	} `json:"aggregations"`
}

func FetchConnectionAnalyticMetricCountAtTime(ctx context.Context, logger *zap.Logger, client opengovernance.Client, metricIDs []string, integrationTypes []integration.Type, connectionIDs, resourceCollections []string, t time.Time, size int) (map[string]CountWithTime, error) {
	idx := resource.AnalyticsIntegrationSummaryIndex
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
	includeIntegrationTypeMap := make(map[string]bool)
	for _, integrationType := range integrationTypes {
		includeIntegrationTypeMap[integrationType.String()] = true
	}
	includeResourceCollectionMap := make(map[string]bool)
	for _, resourceCollection := range resourceCollections {
		idx = resource.ResourceCollectionsAnalyticsIntegrationSummaryIndex
		includeResourceCollectionMap[resourceCollection] = true
	}

	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	query := string(b)

	logger.Info("FetchConnectionAnalyticMetricCountAtTime", zap.String("query", query), zap.String("index", idx))

	var response FetchConnectionAnalyticMetricCountAtTimeResponse
	err = client.Search(ctx, idx, query, &response)
	if err != nil {
		logger.Error("FetchConnectionAnalyticMetricCountAtTime", zap.Error(err), zap.String("query", query), zap.String("index", idx))
		return nil, err
	}

	result := make(map[string]CountWithTime)
	for _, metricBucket := range response.Aggregations.MetricGroup.Buckets {
		for _, hit := range metricBucket.Latest.Hits.Hits {
			handleConnResults := func(connResults resource.IntegrationMetricTrendSummaryResult) {
				for _, connectionResults := range connResults.Integrations {
					if (len(connectionIDs) > 0 && !includeConnectionMap[connectionResults.IntegrationID]) ||
						(len(integrationTypes) > 0 && !includeIntegrationTypeMap[connectionResults.IntegrationType.String()]) {
						continue
					}
					v := result[hit.Source.MetricID]
					v.Count += connectionResults.ResourceCount
					if v.Time.Before(time.UnixMilli(hit.Source.EvaluatedAt)) {
						v.Time = time.UnixMilli(hit.Source.EvaluatedAt)
					}
					result[hit.Source.MetricID] = v
				}
			}

			if len(resourceCollections) > 0 {
				for rcId, rcResult := range hit.Source.ResourceCollections {
					if !includeResourceCollectionMap[rcId] {
						continue
					}
					handleConnResults(rcResult)
				}
			} else if hit.Source.Integrations != nil {
				handleConnResults(*hit.Source.Integrations)
			} else {
				logger.Warn("FetchConnectionAnalyticMetricCountAtTime", zap.String("error", "no connections or resource collections found"))
				return nil, errors.New("no connections or resource collections found")
			}
		}
	}

	return result, nil
}

type FetchIntegrationTypeAnalyticMetricCountAtTimeResponse struct {
	Aggregations struct {
		MetricGroup struct {
			Buckets []struct {
				Key    string `json:"key"`
				Latest struct {
					Hits struct {
						Hits []struct {
							Source resource.IntegrationTypeMetricTrendSummary `json:"_source"`
						} `json:"hits"`
					} `json:"hits"`
				} `json:"latest"`
			} `json:"buckets"`
		} `json:"metric_group"`
	} `json:"aggregations"`
}

func FetchIntegrationTypeAnalyticMetricCountAtTime(ctx context.Context, logger *zap.Logger, client opengovernance.Client,
	metricIDs []string, integrationTypes []integration.Type, resourceCollections []string, t time.Time, size int,
) (map[string]CountWithTime, error) {
	idx := resource.AnalyticsIntegrationTypeSummaryIndex
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

	includeIntegrationTypeMap := make(map[string]bool)
	for _, integrationType := range integrationTypes {
		includeIntegrationTypeMap[integrationType.String()] = true
	}
	includeResourceCollectionMap := make(map[string]bool)
	for _, resourceCollection := range resourceCollections {
		idx = resource.ResourceCollectionsAnalyticsIntegrationTypeSummaryIndex
		includeResourceCollectionMap[resourceCollection] = true
	}

	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	query := string(b)

	logger.Info("FetchIntegrationTypeAnalyticMetricCountAtTime", zap.String("query", query), zap.String("index", idx))

	var response FetchIntegrationTypeAnalyticMetricCountAtTimeResponse
	err = client.Search(ctx, idx, query, &response)
	if err != nil {
		logger.Error("FetchIntegrationTypeAnalyticMetricCountAtTime", zap.Error(err), zap.String("query", query), zap.String("index", idx))
		return nil, err
	}

	result := make(map[string]CountWithTime)
	for _, metricBucket := range response.Aggregations.MetricGroup.Buckets {
		for _, hit := range metricBucket.Latest.Hits.Hits {
			handleConnResults := func(connResults resource.IntegrationTypeMetricTrendSummaryResult) {
				for _, integrationResults := range connResults.IntegrationTypes {
					if len(integrationTypes) > 0 && !includeIntegrationTypeMap[integrationResults.IntegrationType.String()] {
						continue
					}
					v := result[hit.Source.MetricID]
					v.Count += integrationResults.ResourceCount
					if v.Time.Before(time.UnixMilli(hit.Source.EvaluatedAt)) {
						v.Time = time.UnixMilli(hit.Source.EvaluatedAt)
					}
					result[hit.Source.MetricID] = v
				}
			}

			if len(resourceCollections) > 0 {
				for rcId, rcResult := range hit.Source.ResourceCollections {
					if !includeResourceCollectionMap[rcId] {
						continue
					}
					handleConnResults(rcResult)
				}
			} else if hit.Source.IntegrationTypes != nil {
				handleConnResults(*hit.Source.IntegrationTypes)
			} else {
				logger.Warn("FetchIntegrationTypeAnalyticMetricCountAtTime", zap.String("error", "no integration types or resource collections found"))
				return nil, errors.New("no integration types or resource collections found")
			}
		}
	}
	return result, nil
}

func FetchPerResourceCollectionIntegrationTypeAnalyticMetricCountAtTime(ctx context.Context, logger *zap.Logger, client opengovernance.Client,
	metricIDs []string, integrationTypes []integration.Type, resourceCollections []string, t time.Time, size int,
) (map[string]map[string]CountWithTime, error) {
	idx := resource.ResourceCollectionsAnalyticsIntegrationTypeSummaryIndex
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

	includeIntegrationTypeMap := make(map[string]bool)
	for _, integrationType := range integrationTypes {
		includeIntegrationTypeMap[integrationType.String()] = true
	}
	includeResourceCollectionMap := make(map[string]bool)
	for _, resourceCollection := range resourceCollections {
		idx = resource.ResourceCollectionsAnalyticsIntegrationTypeSummaryIndex
		includeResourceCollectionMap[resourceCollection] = true
	}

	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	query := string(b)

	logger.Info("FetchPerResourceCollectionIntegrationTypeAnalyticMetricCountAtTime", zap.String("query", query), zap.String("index", idx))

	var response FetchIntegrationTypeAnalyticMetricCountAtTimeResponse
	err = client.Search(ctx, idx, query, &response)
	if err != nil {
		logger.Error("FetchPerResourceCollectionIntegrationTypeAnalyticMetricCountAtTime", zap.Error(err), zap.String("query", query), zap.String("index", idx))
		return nil, err
	}

	result := make(map[string]map[string]CountWithTime)

	for _, metricBucket := range response.Aggregations.MetricGroup.Buckets {
		for _, hit := range metricBucket.Latest.Hits.Hits {
			for rcId, rcResult := range hit.Source.ResourceCollections {
				if len(resourceCollections) > 0 && !includeResourceCollectionMap[rcId] {
					continue
				}
				if _, ok := result[rcId]; !ok {
					result[rcId] = make(map[string]CountWithTime)
				}
				for _, integrationResults := range rcResult.IntegrationTypes {
					if len(integrationTypes) > 0 && !includeIntegrationTypeMap[integrationResults.IntegrationType.String()] {
						continue
					}
					v := result[rcId][hit.Source.MetricID]
					v.Count += integrationResults.ResourceCount
					if v.Time.Before(time.UnixMilli(hit.Source.EvaluatedAt)) {
						v.Time = time.UnixMilli(hit.Source.EvaluatedAt)
					}
					result[rcId][hit.Source.MetricID] = v
				}
			}
		}
	}

	return result, nil
}

type DatapointWithFailures struct {
	Cost                       float64
	CostStacked                map[string]float64
	Count                      int
	CountStacked               map[string]int
	TotalSuccessfulConnections int64
	TotalConnections           int64

	connectionSuccess      map[string]bool
	integrationTypeSuccess map[string]int64
	integrationTypeTotal   map[string]int64
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
									Source resource.IntegrationMetricTrendSummary `json:"_source"`
								} `json:"hits"`
							} `json:"hits"`
						} `json:"latest"`
					} `json:"buckets"`
				} `json:"evaluated_at_range_group"`
			} `json:"buckets"`
		} `json:"metric_group"`
	} `json:"aggregations"`
}

func FetchConnectionMetricTrendSummaryPage(ctx context.Context, logger *zap.Logger, client opengovernance.Client, connectionIDs []string, integrationTypes []integration.Type, metricIDs, resourceCollections []string, startTime, endTime time.Time, datapointCount, size int) (map[int]DatapointWithFailures, error) {
	idx := resource.AnalyticsIntegrationSummaryIndex
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
	includeIntegrationTypeMap := make(map[string]bool)
	for _, integrationType := range integrationTypes {
		includeIntegrationTypeMap[integrationType.String()] = true
	}
	includeResourceCollectionMap := make(map[string]bool)
	for _, resourceCollection := range resourceCollections {
		idx = resource.ResourceCollectionsAnalyticsIntegrationSummaryIndex
		includeResourceCollectionMap[resourceCollection] = true
	}

	hits := make(map[int]DatapointWithFailures)

	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}
	query := string(b)

	logger.Info("FetchConnectionMetricTrendSummaryPage", zap.String("query", query), zap.String("index", idx))

	var response ConnectionMetricTrendSummaryQueryResponse
	err = client.Search(ctx, idx, query, &response)
	if err != nil {
		logger.Error("FetchConnectionMetricTrendSummaryPage", zap.Error(err), zap.String("query", query), zap.String("index", idx))
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
						CountStacked:      map[string]int{},
					}
				}

				handleConnResults := func(connResults resource.IntegrationMetricTrendSummaryResult) {
					for _, connectionResults := range connResults.Integrations {
						if (len(connectionIDs) > 0 && !includeConnectionMap[connectionResults.IntegrationID]) ||
							(len(integrationTypes) > 0 && !includeIntegrationTypeMap[connectionResults.IntegrationType.String()]) {
							continue
						}
						v.Count += connectionResults.ResourceCount
						v.CountStacked[metricBucket.Key] += connectionResults.ResourceCount
						if _, ok := v.connectionSuccess[connectionResults.IntegrationID]; !ok {
							v.connectionSuccess[connectionResults.IntegrationID] = connectionResults.IsJobSuccessful
						} else {
							v.connectionSuccess[connectionResults.IntegrationID] = v.connectionSuccess[connectionResults.IntegrationID] && connectionResults.IsJobSuccessful
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
				} else if hit.Source.Integrations != nil {
					handleConnResults(*hit.Source.Integrations)
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

type IntegrationTypeMetricTrendSummaryQueryResponse struct {
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
									Source resource.IntegrationTypeMetricTrendSummary `json:"_source"`
								} `json:"hits"`
							} `json:"hits"`
						} `json:"latest"`
					} `json:"buckets"`
				} `json:"evaluated_at_range_group"`
			} `json:"buckets"`
		} `json:"metric_group"`
	} `json:"aggregations"`
}

func FetchIntegrationTypeMetricTrendSummaryPage(ctx context.Context, logger *zap.Logger, client opengovernance.Client, integrationTypes []integration.Type, metricIDs []string, resourceCollections []string, startTime time.Time, endTime time.Time, datapointCount int, size int) (map[int]DatapointWithFailures, error) {
	idx := resource.AnalyticsIntegrationTypeSummaryIndex
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

	includeintegrationTypeMap := make(map[string]bool)
	for _, integrationType := range integrationTypes {
		includeintegrationTypeMap[integrationType.String()] = true
	}
	includeResourceCollectionMap := make(map[string]bool)
	for _, resourceCollection := range resourceCollections {
		idx = resource.ResourceCollectionsAnalyticsIntegrationTypeSummaryIndex
		includeResourceCollectionMap[resourceCollection] = true
	}

	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	query := string(b)

	logger.Info("FetchIntegrationTypeMetricTrendSummaryPage", zap.String("query", query), zap.String("index", idx))

	var response IntegrationTypeMetricTrendSummaryQueryResponse
	err = client.Search(ctx, idx, query, &response)
	if err != nil {
		logger.Error("FetchIntegrationTypeMetricTrendSummaryPage", zap.Error(err), zap.String("query", query), zap.String("index", idx))
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
						integrationTypeTotal:   map[string]int64{},
						integrationTypeSuccess: map[string]int64{},
						CountStacked:           map[string]int{},
					}
					hits[rangeKey] = v
				}

				handleConnResults := func(connResults resource.IntegrationTypeMetricTrendSummaryResult) {
					for _, integrationResults := range connResults.IntegrationTypes {
						if len(integrationTypes) > 0 && !includeintegrationTypeMap[integrationResults.IntegrationType.String()] {
							continue
						}
						v.Count += integrationResults.ResourceCount
						v.CountStacked[metricBucket.Key] += integrationResults.ResourceCount
						v.integrationTypeTotal[integrationResults.IntegrationType.String()] = max(v.integrationTypeTotal[integrationResults.IntegrationType.String()], integrationResults.TotalIntegrationTypes)
						if _, ok := v.integrationTypeSuccess[integrationResults.IntegrationType.String()]; !ok {
							v.integrationTypeSuccess[integrationResults.IntegrationType.String()] = integrationResults.TotalSuccessfulIntegrationTypes
						} else {
							v.integrationTypeSuccess[integrationResults.IntegrationType.String()] = min(v.integrationTypeSuccess[integrationResults.IntegrationType.String()], integrationResults.TotalSuccessfulIntegrationTypes)
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
				} else if hit.Source.IntegrationTypes != nil {
					handleConnResults(*hit.Source.IntegrationTypes)
				} else {
					return nil, errors.New("no integration types or resource collections found")
				}
				hits[rangeKey] = v
			}
		}
	}

	for k, v := range hits {
		for _, total := range v.integrationTypeTotal {
			v.TotalConnections += total
		}
		for _, total := range v.integrationTypeSuccess {
			v.TotalSuccessfulConnections += total
		}
		v.integrationTypeSuccess = nil
		v.integrationTypeTotal = nil
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
							Source resource.IntegrationMetricTrendSummary `json:"_source"`
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

func FetchConnectionAnalyticsResourcesCountAtTime(ctx context.Context, logger *zap.Logger, client opengovernance.Client, integrationTypes []integration.Type, connectionIDs []string,
	resourceCollections []string, metricIDs []string, t time.Time, size int,
) (map[string]FetchConnectionAnalyticsResourcesCountAtTimeReturnValue, error) {
	idx := resource.AnalyticsIntegrationSummaryIndex
	if len(resourceCollections) > 0 {
		idx = resource.ResourceCollectionsAnalyticsIntegrationSummaryIndex
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
	includeIntegrationTypeMap := make(map[string]bool)
	for _, integrationType := range integrationTypes {
		includeIntegrationTypeMap[integrationType.String()] = true
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

	logger.Info("FetchConnectionAnalyticsResourcesCountAtTime", zap.String("query", query), zap.String("index", idx))

	var response FetchConnectionAnalyticsResourcesCountAtTimeResponse
	err = client.Search(
		ctx,
		idx,
		query,
		&response)
	if err != nil {
		logger.Error("FetchConnectionAnalyticsResourcesCountAtTime", zap.Error(err), zap.String("query", query), zap.String("index", idx))
		return nil, err
	}

	result := make(map[string]FetchConnectionAnalyticsResourcesCountAtTimeReturnValue)
	for _, metricBucket := range response.Aggregations.MetricIDGroup.Buckets {
		for _, hit := range metricBucket.Latest.Hits.Hits {
			if len(resourceCollections) > 0 {
				for rcId, rcResult := range hit.Source.ResourceCollections {
					if !includeResourceCollectionMap[rcId] {
						continue
					}
					for _, connectionResults := range rcResult.Integrations {
						if (len(connectionIDs) > 0 && !includeConnectionMap[connectionResults.IntegrationID]) ||
							(len(integrationTypes) > 0 && !includeIntegrationTypeMap[connectionResults.IntegrationType.String()]) {
							continue
						}
						v := result[connectionResults.IntegrationID]
						v.ResourceCountsSum += connectionResults.ResourceCount
						v.LatestEvaluatedAt = max(v.LatestEvaluatedAt, hit.Source.EvaluatedAt)
						result[connectionResults.IntegrationID] = v
					}
				}
			} else {
				for _, connectionResults := range hit.Source.Integrations.Integrations {
					if (len(connectionIDs) > 0 && !includeConnectionMap[connectionResults.IntegrationID]) ||
						(len(integrationTypes) > 0 && !includeIntegrationTypeMap[connectionResults.IntegrationType.String()]) {
						continue
					}
					v := result[connectionResults.IntegrationID]
					v.ResourceCountsSum += connectionResults.ResourceCount
					v.LatestEvaluatedAt = max(v.LatestEvaluatedAt, hit.Source.EvaluatedAt)
					result[connectionResults.IntegrationID] = v
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
									Source resource.IntegrationMetricTrendSummary `json:"_source"`
								} `json:"hits"`
							} `json:"hits"`
						} `json:"latest"`
					} `json:"buckets"`
				} `json:"date_group"`
			} `json:"buckets"`
		} `json:"metric_id_group"`
	} `json:"aggregations"`
}

func FetchAssetTableByDimension(ctx context.Context, logger *zap.Logger, client opengovernance.Client, metricIds []string, granularity inventoryApi.TableGranularityType, dimension inventoryApi.DimensionType, startTime, endTime time.Time) ([]DimensionTrend, error) {
	query := make(map[string]any)
	var filters []any

	index := ""
	switch dimension {
	case inventoryApi.DimensionTypeConnection:
		index = resource.AnalyticsIntegrationSummaryIndex
	case inventoryApi.DimensionTypeMetric:
		index = resource.AnalyticsIntegrationTypeSummaryIndex
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

	logger.Info("FetchAssetTableByDimension", zap.String("query", string(queryJson)), zap.String("index", index))

	var response AssetTableByDimensionQueryResponse
	err = client.Search(ctx, index, string(queryJson), &response)
	if err != nil {
		logger.Error("FetchAssetTableByDimension", zap.Error(err), zap.String("query", string(queryJson)), zap.String("index", index))
		return nil, err
	}

	resultMap := make(map[string]DimensionTrend)
	for _, bucket := range response.Aggregations.MetricIdGroup.Buckets {
		for _, dateBucket := range bucket.DateGroup.Buckets {
			for _, hit := range dateBucket.Latest.Hits.Hits {
				switch dimension {
				case inventoryApi.DimensionTypeConnection:
					for _, connectionResults := range hit.Source.Integrations.Integrations {
						mt, ok := resultMap[connectionResults.IntegrationID]
						if !ok {
							mt = DimensionTrend{
								DimensionID:     connectionResults.IntegrationID,
								DimensionName:   connectionResults.IntegrationName,
								IntegrationType: connectionResults.IntegrationType,
								Trend:           make(map[string]float64),
							}
						}
						mt.Trend[dateBucket.Key] += float64(connectionResults.ResourceCount)
						resultMap[connectionResults.IntegrationID] = mt
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
					mt.Trend[dateBucket.Key] += float64(hit.Source.Integrations.TotalResourceCount)
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

type CountAnalyticsMetricsResponse struct {
	Aggregations struct {
		MetricCount struct {
			Value int `json:"value"`
		} `json:"metric_count"`
		IntegrationCount struct {
			Value int `json:"value"`
		} `json:"integration_count"`
	} `json:"aggregations"`
}

func CountAnalyticsMetrics(ctx context.Context, logger *zap.Logger, client opengovernance.Client) (*CountAnalyticsMetricsResponse, error) {
	query := make(map[string]any)
	query["size"] = 0

	integrationScript := `
String[] res = new String[params['_source']['integrations']['integrations'].length];
for (int i=0; i<params['_source']['integrations']['integrations'].length;++i) { 
  res[i] = params['_source']['integrations']['integrations'][i]['connection_id'];
} 
return res;
`
	query["aggs"] = map[string]any{
		"metric_count": map[string]any{
			"cardinality": map[string]any{
				"field": "metric_id",
			},
		},
		"integration_count": map[string]any{
			"cardinality": map[string]any{
				"script": map[string]any{
					"lang":   "painless",
					"source": integrationScript,
				},
			},
		},
	}

	queryJson, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}

	logger.Info("CountAnalyticsMetrics", zap.String("query", string(queryJson)))

	var response CountAnalyticsMetricsResponse
	err = client.Search(ctx, resource.AnalyticsIntegrationSummaryIndex, string(queryJson), &response)
	if err != nil {
		logger.Error("CountAnalyticsMetrics", zap.Error(err), zap.String("query", string(queryJson)))
		return nil, err
	}

	return &response, nil
}
