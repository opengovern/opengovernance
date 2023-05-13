package es

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/source"

	"gitlab.com/keibiengine/keibi-engine/pkg/insight/es"

	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
)

var MAX_INSIGHTS = 10000

type InsightResultQueryResponse struct {
	Hits         InsightResultQueryHits `json:"hits"`
	Aggregations *struct {
		QueryIDGroup struct {
			Buckets []struct {
				Key         int64 `json:"key"`
				LatestGroup struct {
					Hits InsightResultQueryHits `json:"hits"`
				} `json:"latest_group"`
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

func BuildFindInsightResultsQuery(providerFilter source.Type, sourceIDsFilter []string, uuidFilter *string, startTimeFilter, endTimeFilter *time.Time, queryIDFilter []uint, useHistoricalData bool, useProviderAggregate bool) map[string]any {
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

	if uuidFilter != nil {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"schedule_uuid": {*uuidFilter}},
		})
	}

	if queryIDFilter != nil {
		filters = append(filters, map[string]any{
			"terms": map[string][]uint{"query_id": queryIDFilter},
		})
	}

	if providerFilter != source.Nil {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"provider": {providerFilter.String()}},
		})
	}

	if sourceIDsFilter != nil {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"source_id": sourceIDsFilter},
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

func FetchInsightValueAtTime(client keibi.Client, t time.Time, provider source.Type, sourceIDs []string, insightIds []uint, useHistoricalData bool) (map[uint]es.InsightResource, error) {
	var query map[string]any
	if sourceIDs == nil {
		query = BuildFindInsightResultsQuery(provider, nil, nil, nil, &t, insightIds, useHistoricalData, true)
	} else {
		query = BuildFindInsightResultsQuery(provider, sourceIDs, nil, nil, &t, insightIds, useHistoricalData, false)
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

	result := make(map[uint]es.InsightResource)
	for _, bucket := range response.Aggregations.QueryIDGroup.Buckets {
		if len(bucket.LatestGroup.Hits.Hits) > 0 {
			result[uint(bucket.Key)] = bucket.LatestGroup.Hits.Hits[0].Source
		}
	}
	return result, nil
}

type InsightHistoryResultQueryResponse struct {
	Aggregations struct {
		ScheduleUUIDGroup struct {
			Buckets []struct {
				Key          string `json:"key"`
				QueryIDGroup struct {
					Buckets []struct {
						Key         int64 `json:"key"`
						LatestGroup struct {
							Hits InsightResultQueryHits `json:"hits"`
						} `json:"latest_group"`
					} `json:"buckets"`
				} `json:"query_id_group"`
			}
		} `json:"schedule_uuid_group"`
	} `json:"aggregations"`
}

func FetchInsightAggregatedPerQueryValuesBetweenTimes(client keibi.Client, startTime time.Time, endTime time.Time, provider source.Type, sourceIDs []string, insightIds []uint) (map[uint][]es.InsightResource, error) {
	var query map[string]any
	if sourceIDs == nil {
		query = BuildFindInsightResultsQuery(provider, nil, nil, &startTime, &endTime, insightIds, true, true)
	} else {
		query = BuildFindInsightResultsQuery(provider, sourceIDs, nil, &startTime, &endTime, insightIds, true, false)
	}
	query["size"] = 0
	delete(query, "sort")
	query["aggs"] = map[string]any{
		"schedule_uuid_group": map[string]any{
			"terms": map[string]any{
				"field": "schedule_uuid",
				"size":  MAX_INSIGHTS,
			},
			"aggs": map[string]any{
				"query_id_group": map[string]any{
					"terms": map[string]any{
						"field": "query_id",
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
	var response InsightHistoryResultQueryResponse
	err = client.Search(context.Background(), es.InsightsIndex, string(queryJson), &response)
	if err != nil {
		return nil, err
	}

	result := make(map[uint][]es.InsightResource)
	for _, uuidBucket := range response.Aggregations.ScheduleUUIDGroup.Buckets {
		for _, bucket := range uuidBucket.QueryIDGroup.Buckets {
			if len(bucket.LatestGroup.Hits.Hits) > 0 {
				if _, ok := result[uint(bucket.Key)]; !ok {
					result[uint(bucket.Key)] = []es.InsightResource{}
				}
				result[uint(bucket.Key)] = append(result[uint(bucket.Key)], bucket.LatestGroup.Hits.Hits[0].Source)
			}
		}
	}
	for _, res := range result {
		sort.Slice(res, func(i, j int) bool {
			return res[i].ExecutedAt < res[j].ExecutedAt
		})
	}
	return result, nil
}

func FetchInsightRecordByQueryAndJobId(client keibi.Client, queryId uint, jobId uint) (*es.InsightResource, error) {
	boolQuery := map[string]any{}
	var filters []any
	filters = append(filters, map[string]any{
		"terms": map[string][]string{"resource_type": {string(es.InsightResourceHistory), string(es.InsightResourceProviderHistory)}},
	})
	filters = append(filters, map[string]any{
		"terms": map[string][]uint{"query_id": {queryId}},
	})
	filters = append(filters, map[string]any{
		"terms": map[string][]uint{"job_id": {jobId}},
	})
	boolQuery["filter"] = filters

	res := make(map[string]any)
	res["size"] = 1
	res["query"] = map[string]any{
		"bool": boolQuery,
	}

	queryJson, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}
	fmt.Println("query=", string(queryJson), "index=", es.InsightsIndex)
	var response InsightResultQueryResponse
	err = client.Search(context.Background(), es.InsightsIndex, string(queryJson), &response)
	if err != nil {
		return nil, err
	}

	for _, hit := range response.Hits.Hits {
		return &hit.Source, nil
	}
	return nil, nil
}
