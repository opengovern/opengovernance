package es

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/kaytu-io/kaytu-util/pkg/source"

	"github.com/kaytu-io/kaytu-engine/pkg/insight/es"

	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
)

var MAX_INSIGHTS = 10000

type InsightResultQueryResponse struct {
	Hits         InsightResultQueryHits `json:"hits"`
	Aggregations *struct {
		InsightIDGroup struct {
			Buckets []struct {
				Key           int64 `json:"key"`
				SourceIDGroup struct {
					Buckets []struct {
						Key         string `json:"key"`
						LatestGroup struct {
							Hits InsightResultQueryHits `json:"hits"`
						} `json:"latest_group"`
					} `json:"buckets"`
				} `json:"source_id_group"`
			} `json:"buckets"`
		} `json:"insight_id_group"`
	} `json:"aggregations,omitempty"`
}
type InsightResultQueryHits struct {
	Total kaytu.SearchTotal `json:"total"`
	Hits  []struct {
		ID      string             `json:"_id"`
		Score   float64            `json:"_score"`
		Index   string             `json:"_index"`
		Type    string             `json:"_type"`
		Version int64              `json:"_version,omitempty"`
		Source  es.InsightResource `json:"_source"`
		Sort    []any              `json:"sort"`
	} `json:"hits"`
}

func BuildFindInsightResultsQuery(
	connectors []source.Type,
	startTimeFilter, endTimeFilter *time.Time,
	insightIDFilter []uint,
	useHistoricalData bool) map[string]any {
	boolQuery := map[string]any{}
	var filters []any

	var resourceType es.InsightResourceType
	if useHistoricalData {
		resourceType = es.InsightResourceProviderHistory
	} else {
		resourceType = es.InsightResourceProviderLast
	}

	filters = append(filters, map[string]any{
		"terms": map[string][]string{"resource_type": {string(resourceType)}},
	})

	if insightIDFilter != nil {
		filters = append(filters, map[string]any{
			"terms": map[string][]uint{"insight_id": insightIDFilter},
		})
	}

	if len(connectors) > 0 {
		connectorsStr := make([]string, 0, len(connectors))
		for _, connector := range connectors {
			connectorsStr = append(connectorsStr, connector.String())
		}
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"provider": connectorsStr},
		})
	}

	if startTimeFilter != nil || endTimeFilter != nil {
		rangeFilter := map[string]any{
			"executed_at": map[string]any{},
		}
		if startTimeFilter != nil {
			rangeFilter["executed_at"].(map[string]any)["gte"] = startTimeFilter.UnixMilli()
		}
		if endTimeFilter != nil {
			rangeFilter["executed_at"].(map[string]any)["lte"] = endTimeFilter.UnixMilli()
		}
		filters = append(filters, map[string]any{
			"range": rangeFilter,
		})
	}

	boolQuery["filter"] = filters

	res := make(map[string]any)
	res["size"] = MAX_INSIGHTS
	res["sort"] = []map[string]any{
		{
			"_id": "asc",
		},
	}

	if len(boolQuery) > 0 {
		res["query"] = map[string]any{
			"bool": boolQuery,
		}
	}

	return res
}

func FetchInsightValueAtTime(client kaytu.Client, t time.Time, connectors []source.Type, connectionIDs []string, insightIds []uint, useHistoricalData bool) (map[uint][]es.InsightResource, error) {
	query := BuildFindInsightResultsQuery(connectors, nil, &t, insightIds, useHistoricalData)
	query["size"] = 0
	delete(query, "sort")
	query["aggs"] = map[string]any{
		"insight_id_group": map[string]any{
			"terms": map[string]any{
				"field": "insight_id",
				"size":  MAX_INSIGHTS,
			},
			"aggs": map[string]any{
				"source_id_group": map[string]any{
					"terms": map[string]any{
						"field": "source_id",
						"size":  MAX_INSIGHTS,
					},
					"aggs": map[string]any{
						"latest_group": map[string]any{
							"top_hits": map[string]any{
								"size": 1,
								"sort": []map[string]any{
									{
										"executed_at": map[string]any{
											"order": "desc",
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

	isStack := false
	if len(connectionIDs) > 0 {
		if strings.HasPrefix(connectionIDs[0], "stack-") {
			isStack = true
		}
	}

	queryJson, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}
	var response InsightResultQueryResponse

	if isStack {
		fmt.Println("query=", string(queryJson), "index=", es.StacksInsightsIndex)
		err = client.Search(context.Background(), es.StacksInsightsIndex, string(queryJson), &response)
		if err != nil {
			return nil, err
		}
	} else {
		fmt.Println("query=", string(queryJson), "index=", es.InsightsIndex)
		err = client.Search(context.Background(), es.InsightsIndex, string(queryJson), &response)
		if err != nil {
			return nil, err
		}
	}

	result := make(map[uint][]es.InsightResource)
	if response.Aggregations == nil {
		return nil, nil
	}
	for _, insightIdBucket := range response.Aggregations.InsightIDGroup.Buckets {
		for _, sourceIdBucket := range insightIdBucket.SourceIDGroup.Buckets {
			for _, hit := range sourceIdBucket.LatestGroup.Hits.Hits {
				insightResult := hit.Source
				if len(connectionIDs) > 0 {
					insightResult.Result = 0
					insightResult.PerConnectionCount = make(map[string]int64)
					for _, connectionID := range connectionIDs {
						if count, ok := hit.Source.PerConnectionCount[connectionID]; ok {
							insightResult.Result += count
							insightResult.PerConnectionCount[connectionID] = count
						}
					}
				}
				result[uint(insightIdBucket.Key)] = append(result[uint(insightIdBucket.Key)], insightResult)
			}
		}
	}
	return result, nil
}

func FetchInsightValueAfter(client kaytu.Client, t time.Time, connectors []source.Type, connectionIDs []string, insightIds []uint) (map[uint][]es.InsightResource, error) {
	query := BuildFindInsightResultsQuery(connectors, &t, nil, insightIds, true)
	query["size"] = 0
	delete(query, "sort")

	query["aggs"] = map[string]any{
		"insight_id_group": map[string]any{
			"terms": map[string]any{
				"field": "insight_id",
				"size":  MAX_INSIGHTS,
			},
			"aggs": map[string]any{
				"source_id_group": map[string]any{
					"terms": map[string]any{
						"field": "source_id",
						"size":  MAX_INSIGHTS,
					},
					"aggs": map[string]any{
						"latest_group": map[string]any{
							"top_hits": map[string]any{
								"size": 1,
								"sort": []map[string]any{
									{
										"executed_at": map[string]any{
											"order": "asc",
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

	queryJson, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}

	isStack := false
	if len(connectionIDs) > 0 {
		if strings.HasPrefix(connectionIDs[0], "stack-") {
			isStack = true
		}
	}

	var response InsightResultQueryResponse

	if isStack {
		fmt.Println("query=", string(queryJson), "index=", es.StacksInsightsIndex)
		err = client.Search(context.Background(), es.StacksInsightsIndex, string(queryJson), &response)
		if err != nil {
			return nil, err
		}
	} else {
		fmt.Println("query=", string(queryJson), "index=", es.InsightsIndex)
		err = client.Search(context.Background(), es.InsightsIndex, string(queryJson), &response)
		if err != nil {
			return nil, err
		}
	}

	result := make(map[uint][]es.InsightResource)
	if response.Aggregations == nil {
		return nil, nil
	}
	for _, insightIdBucket := range response.Aggregations.InsightIDGroup.Buckets {
		for _, sourceIdBucket := range insightIdBucket.SourceIDGroup.Buckets {
			for _, hit := range sourceIdBucket.LatestGroup.Hits.Hits {
				insightResult := hit.Source
				if len(connectionIDs) > 0 {
					insightResult.Result = 0
					insightResult.PerConnectionCount = make(map[string]int64)
					for _, connectionID := range connectionIDs {
						if count, ok := hit.Source.PerConnectionCount[connectionID]; ok {
							insightResult.Result += count
							insightResult.PerConnectionCount[connectionID] = count
						}
					}
				}
				result[uint(insightIdBucket.Key)] = append(result[uint(insightIdBucket.Key)], insightResult)
			}
		}
		sort.Slice(result[uint(insightIdBucket.Key)], func(i, j int) bool {
			return result[uint(insightIdBucket.Key)][i].ExecutedAt < result[uint(insightIdBucket.Key)][j].ExecutedAt
		})
	}
	return result, nil
}

type InsightHistoryResultQueryResponse struct {
	Aggregations struct {
		InsightIDGroup struct {
			Buckets []struct {
				Key                  int64 `json:"key"`
				ExecutedAtRangeGroup struct {
					Buckets []struct {
						From          float64 `json:"from"`
						To            float64 `json:"to"`
						SourceIDGroup struct {
							Buckets []struct {
								Key         string `json:"key"`
								LatestGroup struct {
									Hits InsightResultQueryHits `json:"hits"`
								} `json:"latest_group"`
							} `json:"buckets"`
						} `json:"source_id_group"`
					} `json:"buckets"`
				} `json:"executed_at_range_group"`
			} `json:"buckets"`
		} `json:"insight_id_group"`
	} `json:"aggregations"`
}

func FetchInsightAggregatedPerQueryValuesBetweenTimes(client kaytu.Client, startTime time.Time, endTime time.Time, datapointCount int, connectors []source.Type, connectionIDs []string, insightIds []uint) (map[uint]map[int][]es.InsightResource, error) {
	query := BuildFindInsightResultsQuery(connectors, &startTime, &endTime, insightIds, true)
	query["size"] = 0
	delete(query, "sort")

	startTimeUnixMilli := startTime.UnixMilli()
	endTimeUnixMilli := endTime.UnixMilli()
	step := int(math.Ceil(float64(endTimeUnixMilli-startTimeUnixMilli) / float64(datapointCount)))
	ranges := make([]map[string]any, 0, datapointCount)
	for i := 0; i < datapointCount; i++ {
		ranges = append(ranges, map[string]any{
			"from": startTimeUnixMilli + int64(i*step),
			"to":   startTimeUnixMilli + int64((i+1)*step),
		})
	}

	query["aggs"] = map[string]any{
		"insight_id_group": map[string]any{
			"terms": map[string]any{
				"field": "insight_id",
				"size":  MAX_INSIGHTS,
			},
			"aggs": map[string]any{
				"executed_at_range_group": map[string]any{
					"range": map[string]any{
						"field":  "executed_at",
						"ranges": ranges,
					},
					"aggs": map[string]any{
						"source_id_group": map[string]any{
							"terms": map[string]any{
								"field": "source_id",
								"size":  MAX_INSIGHTS,
							},
							"aggs": map[string]any{
								"latest_group": map[string]any{
									"top_hits": map[string]any{
										"size": 1,
										"sort": []map[string]any{
											{
												"executed_at": map[string]any{
													"order": "desc",
												},
											},
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

	queryJson, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}

	isStack := false
	if len(connectionIDs) > 0 {
		if strings.HasPrefix(connectionIDs[0], "stack-") {
			isStack = true
		}
	}

	var response InsightHistoryResultQueryResponse
	if isStack {
		fmt.Println("query=", string(queryJson), "index=", es.StacksInsightsIndex)
		err = client.Search(context.Background(), es.StacksInsightsIndex, string(queryJson), &response)
		if err != nil {
			return nil, err
		}
	} else {
		fmt.Println("query=", string(queryJson), "index=", es.InsightsIndex)
		err = client.Search(context.Background(), es.InsightsIndex, string(queryJson), &response)
		if err != nil {
			return nil, err
		}
	}

	result := make(map[uint]map[int][]es.InsightResource)
	for _, insightIDBucket := range response.Aggregations.InsightIDGroup.Buckets {
		if _, ok := result[uint(insightIDBucket.Key)]; !ok {
			result[uint(insightIDBucket.Key)] = make(map[int][]es.InsightResource)
		}
		for _, rangeBucket := range insightIDBucket.ExecutedAtRangeGroup.Buckets {
			rangeBucketKey := int(rangeBucket.To) / 1000 // convert to seconds
			//rangeBucketKey := int((rangeBucket.From+rangeBucket.To)/2) / 1000 // convert to seconds
			for _, sourceIDBucket := range rangeBucket.SourceIDGroup.Buckets {
				for _, hit := range sourceIDBucket.LatestGroup.Hits.Hits {
					insightResult := hit.Source
					if len(connectionIDs) > 0 {
						insightResult.Result = 0
						insightResult.PerConnectionCount = make(map[string]int64)
						for _, connectionID := range connectionIDs {
							if count, ok := hit.Source.PerConnectionCount[connectionID]; ok {
								insightResult.Result += count
								insightResult.PerConnectionCount[connectionID] = count
							}
						}
					}
					result[uint(insightIDBucket.Key)][rangeBucketKey] = append(result[uint(insightIDBucket.Key)][rangeBucketKey], insightResult)
				}
			}
			sort.Slice(result[uint(insightIDBucket.Key)][rangeBucketKey], func(i, j int) bool {
				return result[uint(insightIDBucket.Key)][rangeBucketKey][i].ExecutedAt < result[uint(insightIDBucket.Key)][rangeBucketKey][j].ExecutedAt
			})
		}
	}
	return result, nil
}
