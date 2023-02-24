package es

import (
	"context"
	"encoding/json"
	"errors"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"

	"gitlab.com/keibiengine/keibi-engine/pkg/insight/es"

	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
)

var MAX_INSIGHTS = 10000

type InsightResultQueryResponse struct {
	Hits InsightResultQueryHits `json:"hits"`
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

func FindInsightResults(providerFilter *source.Type, sourceIDFilter *string, uuidFilter *string, queryIDFilter *uint, useHistoricalData bool) (string, error) {
	boolQuery := map[string]interface{}{}
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
			"terms": map[string][]uint{"query_id": {*queryIDFilter}},
		})
	}

	if providerFilter != nil {
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
	b, err := json.Marshal(res)
	return string(b), err
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
