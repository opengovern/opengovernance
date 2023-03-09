package es

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/source"

	"gitlab.com/keibiengine/keibi-engine/pkg/insight/es"

	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
)

var MAX_INSIGHTS = 10000

type InsightResultQueryResponse struct {
	Hits         InsightResultQueryHits `json:"hits"`
	Aggregations *struct {
		QueryIDGroup struct {
			Buckets []struct {
				Key        int64 `json:"key"`
				ValueTotal struct {
					Value float64 `json:"value"`
				} `json:"value_total"`
				MinExecutedAt struct {
					Value float64 `json:"value"`
				} `json:"min_executed_at"`
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

type AggregateInsightResult struct {
	InsightID  uint
	Value      int
	ExecutedAt int
}

func BuildFindInsightResultsQuery(providerFilter source.Type, sourceIDFilter, uuidFilter *string, startTimeFilter, endTimeFilter *time.Time, queryIDFilter []uint, useHistoricalData bool) map[string]any {
	boolQuery := map[string]any{}
	var filters []any

	resourceType := es.InsightResourceLast
	if useHistoricalData {
		resourceType = es.InsightResourceHistory
	}

	filters = append(filters, map[string]any{
		"terms": map[string][]string{"resource_type": {resourceType}},
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

	if sourceIDFilter != nil {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"source_id": {*sourceIDFilter}},
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

func FindInsightResultUUID(client keibi.Client, executedAt int64) (string, error) {
	boolQuery := map[string]any{}
	var filters []any
	filters = append(filters, map[string]any{
		"terms": map[string][]string{"resource_type": {es.InsightResourceHistory}},
	})

	filters = append(filters, map[string]any{
		"range": map[string]any{"executed_at": map[string]int64{"lte": executedAt}},
	})

	boolQuery["filter"] = filters

	res := make(map[string]any)
	res["size"] = 1
	res["sort"] = []map[string]any{
		{
			"executed_at": "desc",
			"_id":         "asc",
		},
	}

	if len(boolQuery) > 0 {
		res["query"] = map[string]any{
			"bool": boolQuery,
		}
	}
	b, err := json.Marshal(res)

	var response InsightResultQueryResponse
	err = client.Search(context.Background(), es.InsightsIndex,
		string(b), &response)
	if err != nil {
		return "", err
	}

	for _, hit := range response.Hits.Hits {
		return hit.Source.ScheduleUUID, nil
	}
	return "", errors.New("insight not found")
}

func FetchInsightValuesAtTime(client keibi.Client, t time.Time, provider source.Type, sourceID *string, insightIds []uint, useHistoricalData bool) ([]es.InsightResource, error) {
	lastId, err := FindInsightResultUUID(client, t.UnixMilli())
	if err != nil {
		return nil, err
	}

	query := BuildFindInsightResultsQuery(provider, sourceID, &lastId, nil, nil, insightIds, useHistoricalData)
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

	result := make([]es.InsightResource, 0, len(response.Hits.Hits))

	for _, hit := range response.Hits.Hits {
		result = append(result, hit.Source)
	}
	return result, nil
}

func FetchInsightAggregatedPerQueryValueAtTime(client keibi.Client, t time.Time, provider source.Type, sourceID *string, insightIds []uint, useHistoricalData bool) (map[uint]AggregateInsightResult, error) {
	lastId, err := FindInsightResultUUID(client, t.UnixMilli())
	if err != nil {
		return nil, err
	}

	query := BuildFindInsightResultsQuery(provider, sourceID, &lastId, nil, nil, insightIds, useHistoricalData)
	query["size"] = 0
	query["aggs"] = map[string]any{
		"query_id_group": map[string]any{
			"terms": map[string]any{
				"field": "query_id",
				"size":  MAX_INSIGHTS,
			},
			"aggs": map[string]any{
				"value_total": map[string]any{
					"sum": map[string]any{
						"field": "result",
					},
				},
				"min_executed_at": map[string]any{
					"min": map[string]any{
						"field": "executed_at",
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

	result := make(map[uint]AggregateInsightResult)
	for _, bucket := range response.Aggregations.QueryIDGroup.Buckets {
		result[uint(bucket.Key)] = AggregateInsightResult{
			InsightID:  uint(bucket.Key),
			Value:      int(bucket.ValueTotal.Value),
			ExecutedAt: int(bucket.MinExecutedAt.Value),
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
						Key        int64 `json:"key"`
						ValueTotal struct {
							Value float64 `json:"value"`
						} `json:"value_total"`
						MinExecutedAt struct {
							Value float64 `json:"value"`
						} `json:"min_executed_at"`
					} `json:"buckets"`
				} `json:"query_id_group"`
			}
		} `json:"schedule_uuid_group"`
	} `json:"aggregations"`
}

func FetchInsightAggregatedPerQueryValuesBetweenTimes(client keibi.Client, startTime time.Time, endTime time.Time, provider source.Type, sourceID *string, insightIds []uint) (map[uint][]AggregateInsightResult, error) {
	query := BuildFindInsightResultsQuery(provider, sourceID, nil, &startTime, &endTime, insightIds, true)
	query["size"] = 0
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
						"value_total": map[string]any{
							"sum": map[string]any{
								"field": "result",
							},
						},
						"min_executed_at": map[string]any{
							"min": map[string]any{
								"field": "executed_at",
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

	result := make(map[uint][]AggregateInsightResult)
	for _, uuidBucket := range response.Aggregations.ScheduleUUIDGroup.Buckets {
		for _, bucket := range uuidBucket.QueryIDGroup.Buckets {
			if _, ok := result[uint(bucket.Key)]; !ok {
				result[uint(bucket.Key)] = []AggregateInsightResult{}
			}
			result[uint(bucket.Key)] = append(result[uint(bucket.Key)], AggregateInsightResult{
				InsightID:  uint(bucket.Key),
				Value:      int(bucket.ValueTotal.Value),
				ExecutedAt: int(bucket.MinExecutedAt.Value),
			})
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
		"terms": map[string][]string{"resource_type": {es.InsightResourceHistory}},
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
