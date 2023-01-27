package es

import (
	"encoding/json"

	"gitlab.com/keibiengine/keibi-engine/pkg/insight/es"

	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
)

var MAX_INSIGHTS = 1000

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

func FindInsightResults(descriptionFilter *string, labelFilter []string, sourceIDFilter []string) (string, error) {
	boolQuery := map[string]interface{}{}
	var filters []interface{}
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"resource_type": {es.InsightResourceLast}},
	})

	if labelFilter != nil && len(labelFilter) > 0 {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"labels": labelFilter},
		})
	}

	if sourceIDFilter != nil && len(sourceIDFilter) > 0 {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_id": sourceIDFilter},
		})
	}

	boolQuery["filter"] = filters

	if descriptionFilter != nil && len(*descriptionFilter) > 0 {
		boolQuery["must"] = map[string]interface{}{
			"multi_match": map[string]interface{}{
				"fields":    []string{"query", "result"},
				"query":     *descriptionFilter,
				"fuzziness": "AUTO",
			},
		}
	}

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
