package es

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
				Key        string `json:"key"`
				ValueTotal struct {
					Value float64 `json:"value"`
				} `json:"value_total"`
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
	Sort    []interface{}      `json:"sort"`
}

func BuildFindInsightResultsQuery(providerFilter source.Type, sourceIDFilter *string, uuidFilter *string, queryIDFilter []uint, useHistoricalData bool) map[string]any {
	boolQuery := map[string]any{}
	var filters []interface{}

	resourceType := es.InsightResourceLast
	if useHistoricalData {
		resourceType = es.InsightResourceHistory
	}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"resource_type": {resourceType}},
	})

	if uuidFilter != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"schedule_uuid": {*uuidFilter}},
		})
	}

	if queryIDFilter != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]uint{"query_id": queryIDFilter},
		})
	}

	if providerFilter != source.Nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"provider": {providerFilter.String()}},
		})
	}

	if sourceIDFilter != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_id": {*sourceIDFilter}},
		})
	}

	boolQuery["filter"] = filters

	res := make(map[string]interface{})
	res["size"] = MAX_INSIGHTS
	res["sort"] = []map[string]interface{}{
		{
			"_id": "asc",
		},
	}

	if len(boolQuery) > 0 {
		res["query"] = map[string]interface{}{
			"bool": boolQuery,
		}
	}

	return res
}

func FindInsightResultUUID(client keibi.Client, executedAt int64) (string, error) {
	boolQuery := map[string]interface{}{}
	var filters []interface{}
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"resource_type": {es.InsightResourceHistory}},
	})

	filters = append(filters, map[string]interface{}{
		"range": map[string]interface{}{"executed_at": map[string]int64{"lte": executedAt}},
	})

	boolQuery["filter"] = filters

	res := make(map[string]interface{})
	res["size"] = 1
	res["sort"] = []map[string]interface{}{
		{
			"executed_at": "desc",
			"_id":         "asc",
		},
	}

	if len(boolQuery) > 0 {
		res["query"] = map[string]interface{}{
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

func FetchInsightValuesAtTime(client keibi.Client, t time.Time, provider source.Type, sourceID *string, insightIds []uint) ([]es.InsightResource, error) {
	lastId, err := FindInsightResultUUID(client, t.UnixMilli())
	if err != nil {
		return nil, err
	}

	query := BuildFindInsightResultsQuery(provider, sourceID, &lastId, insightIds, true)
	queryJson, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}
	fmt.Println("query=", queryJson, "index=", es.InsightsIndex)
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

func FetchInsightAggregatedPerQueryValuesAtTime(client keibi.Client, t time.Time, provider source.Type, sourceID *string, insightIds []uint) (map[string]int, error) {
	lastId, err := FindInsightResultUUID(client, t.UnixMilli())
	if err != nil {
		return nil, err
	}

	query := BuildFindInsightResultsQuery(provider, sourceID, &lastId, insightIds, true)
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

	result := make(map[string]int)
	for _, bucket := range response.Aggregations.QueryIDGroup.Buckets {
		result[bucket.Key] = int(bucket.ValueTotal.Value)
	}
	return result, nil
}
