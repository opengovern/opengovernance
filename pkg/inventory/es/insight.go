package es

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/kaytu-io/kaytu-util/pkg/source"

	"gitlab.com/keibiengine/keibi-engine/pkg/insight/es"

	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
)

var MAX_INSIGHTS = 10000

type InsightResultQueryResponse struct {
	Hits         InsightResultQueryHits `json:"hits"`
	Aggregations *struct {
		QueryIDGroup struct {
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
		} `json:"query_id_group"`
	} `json:"aggregations,omitempty"`
}
type InsightResultQueryHits struct {
	Total keibi.SearchTotal       `json:"total"`
	Hits  []InsightResultQueryHit `json:"hits"`
}
type InsightResultQueryHit struct {
	ID      string             `json:"_id"`
	Score   float64            `json:"_score"`
	Index   string             `json:"_index"`
	Type    string             `json:"_type"`
	Version int64              `json:"_version,omitempty"`
	Source  es.InsightResource `json:"_source"`
	Sort    []any              `json:"sort"`
}

func BuildFindInsightResultsQuery(connectors []source.Type, connectionIDsFilter []string, startTimeFilter, endTimeFilter *time.Time, queryIDFilter []uint, useHistoricalData, useProviderAggregate bool) map[string]any {
	boolQuery := map[string]any{}
	var filters []any

	var resourceType es.InsightResourceType
	if useHistoricalData {
		resourceType = es.InsightResourceHistory
		if useProviderAggregate {
			resourceType = es.InsightResourceProviderHistory
		}
	} else {
		resourceType = es.InsightResourceLast
		if useProviderAggregate {
			resourceType = es.InsightResourceProviderLast
		}
	}

	filters = append(filters, map[string]any{
		"terms": map[string][]string{"resource_type": {string(resourceType)}},
	})

	if queryIDFilter != nil {
		filters = append(filters, map[string]any{
			"terms": map[string][]uint{"query_id": queryIDFilter},
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

	if connectionIDsFilter != nil {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"source_id": connectionIDsFilter},
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

func FetchInsightValueAtTime(client keibi.Client, t time.Time, connectors []source.Type, connectionIDs []string, insightIds []uint, useHistoricalData bool) (map[uint][]es.InsightResource, error) {
	var query map[string]any
	if len(connectionIDs) == 0 {
		query = BuildFindInsightResultsQuery(connectors, nil, nil, &t, insightIds, useHistoricalData, true)
	} else {
		query = BuildFindInsightResultsQuery(connectors, connectionIDs, nil, &t, insightIds, useHistoricalData, false)
	}
	query["size"] = 0
	delete(query, "sort")
	query["aggs"] = map[string]any{
		"query_id_group": map[string]any{
			"terms": map[string]any{
				"field": "query_id",
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

	queryJson, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}
	fmt.Println("query=", string(queryJson), "index=", es.InsightsIndex)
	var response InsightResultQueryResponse
	err = client.Search(context.Background(), es.InsightsIndex, string(queryJson), &response)
	if err != nil {
		return nil, err
	}

	result := make(map[uint][]es.InsightResource)
	if response.Aggregations == nil {
		return nil, nil
	}
	for _, queryIdBucket := range response.Aggregations.QueryIDGroup.Buckets {
		for _, sourceIdBucket := range queryIdBucket.SourceIDGroup.Buckets {
			for _, hit := range sourceIdBucket.LatestGroup.Hits.Hits {
				result[uint(queryIdBucket.Key)] = append(result[uint(queryIdBucket.Key)], hit.Source)
			}
		}
	}
	return result, nil
}

type InsightHistoryResultQueryResponse struct {
	Aggregations struct {
		QueryIDGroup struct {
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
		} `json:"query_id_group"`
	} `json:"aggregations"`
}

func FetchInsightAggregatedPerQueryValuesBetweenTimes(client keibi.Client, startTime time.Time, endTime time.Time, datapointCount int, connectors []source.Type, connectionIDs []string, insightIds []uint) (map[uint]map[int][]es.InsightResource, error) {
	var query map[string]any
	if len(connectionIDs) == 0 {
		query = BuildFindInsightResultsQuery(connectors, nil, &startTime, &endTime, insightIds, true, true)
	} else {
		query = BuildFindInsightResultsQuery(connectors, connectionIDs, &startTime, &endTime, insightIds, true, false)
	}
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
		"query_id_group": map[string]any{
			"terms": map[string]any{
				"field": "query_id",
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
	fmt.Println("query=", string(queryJson), "index=", es.InsightsIndex)
	var response InsightHistoryResultQueryResponse
	err = client.Search(context.Background(), es.InsightsIndex, string(queryJson), &response)
	if err != nil {
		return nil, err
	}

	result := make(map[uint]map[int][]es.InsightResource)
	for _, queryIDBucket := range response.Aggregations.QueryIDGroup.Buckets {
		if _, ok := result[uint(queryIDBucket.Key)]; !ok {
			result[uint(queryIDBucket.Key)] = make(map[int][]es.InsightResource)
		}
		for _, rangeBucket := range queryIDBucket.ExecutedAtRangeGroup.Buckets {
			rangeBucketKey := int((rangeBucket.From + rangeBucket.To) / 2)
			for _, sourceIDBucket := range rangeBucket.SourceIDGroup.Buckets {
				for _, hit := range sourceIDBucket.LatestGroup.Hits.Hits {
					result[uint(queryIDBucket.Key)][rangeBucketKey] = append(result[uint(queryIDBucket.Key)][rangeBucketKey], hit.Source)
				}
			}
			sort.Slice(result[uint(queryIDBucket.Key)][rangeBucketKey], func(i, j int) bool {
				return result[uint(queryIDBucket.Key)][rangeBucketKey][i].ExecutedAt < result[uint(queryIDBucket.Key)][rangeBucketKey][j].ExecutedAt
			})
		}
	}
	return result, nil
}
