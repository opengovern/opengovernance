package query

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"

	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
	summarizer "gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/types"
)

type FindingAlarmsQueryResponse struct {
	Hits FindingAlarmsQueryHits `json:"hits"`
}
type FindingAlarmsQueryHits struct {
	Total keibi.SearchTotal       `json:"total"`
	Hits  []FindingAlarmsQueryHit `json:"hits"`
}
type FindingAlarmsQueryHit struct {
	ID      string                  `json:"_id"`
	Score   float64                 `json:"_score"`
	Index   string                  `json:"_index"`
	Type    string                  `json:"_type"`
	Version int64                   `json:"_version,omitempty"`
	Source  summarizer.FindingAlarm `json:"_source"`
	Sort    []interface{}           `json:"sort"`
}

func GetLastActiveAlarm(client keibi.Client, resourceID string, controlID string) (*summarizer.FindingAlarm, error) {
	res := make(map[string]interface{})
	var filters []interface{}
	filters = append(filters, map[string]interface{}{
		"term": map[string]string{
			"resource_id": resourceID,
		},
	})

	filters = append(filters, map[string]interface{}{
		"term": map[string]string{
			"control_id": controlID,
		},
	})

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{
			"status": {
				string(types.ComplianceResultALARM),
				string(types.ComplianceResultINFO),
				string(types.ComplianceResultSKIP),
				string(types.ComplianceResultERROR),
			},
		},
	})

	res["size"] = 1
	res["sort"] = []map[string]interface{}{
		{"last_evaluated": "desc"},
	}
	res["query"] = map[string]interface{}{
		"bool": map[string]interface{}{
			"filter": filters,
		},
	}
	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	query := string(b)

	var response FindingAlarmsQueryResponse
	err = client.Search(context.Background(), summarizer.AlarmIndex, query, &response)
	if err != nil {
		return nil, err
	}

	for _, hit := range response.Hits.Hits {
		return &hit.Source, nil
	}
	return nil, nil
}

func GetAlarms(client keibi.Client, resourceID string, controlID string) ([]summarizer.FindingAlarm, error) {
	res := make(map[string]interface{})
	var filters []interface{}
	filters = append(filters, map[string]interface{}{
		"term": map[string]string{
			"resource_id": resourceID,
		},
	})

	filters = append(filters, map[string]interface{}{
		"term": map[string]string{
			"control_id": controlID,
		},
	})

	res["size"] = summarizer.EsFetchPageSize
	res["sort"] = []map[string]interface{}{
		{"last_evaluated": "desc"},
	}
	res["query"] = map[string]interface{}{
		"bool": map[string]interface{}{
			"filter": filters,
		},
	}
	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	query := string(b)

	var response FindingAlarmsQueryResponse
	err = client.Search(context.Background(), summarizer.AlarmIndex, query, &response)
	if err != nil {
		return nil, err
	}

	var alarms []summarizer.FindingAlarm
	for _, hit := range response.Hits.Hits {
		alarms = append(alarms, hit.Source)
	}
	return alarms, nil
}

type AggregationResult struct {
	DocCountErrorUpperBound int      `json:"doc_count_error_upper_bound"`
	SumOtherDocCount        int      `json:"sum_other_doc_count"`
	Buckets                 []Bucket `json:"buckets"`
}
type Bucket struct {
	Key      string `json:"key"`
	DocCount int    `json:"doc_count"`
}

type AlarmTopFieldResponse struct {
	Aggregations AlarmTopFieldAggregations `json:"aggregations"`
}
type AlarmTopFieldAggregations struct {
	FieldFilter AggregationResult `json:"field_filter"`
}

func AlarmTopFieldQuery(client keibi.Client,
	field string,
	provider []source.Type,
	resourceTypeID []string,
	sourceID []uuid.UUID,
	status []types.ComplianceResult,
	benchmarkID []string,
	policyID []string,
	severity []string,
	size int,
) (*AlarmTopFieldResponse, error) {
	terms := make(map[string]interface{})

	if len(benchmarkID) > 0 {
		terms["benchmarkID"] = benchmarkID
	}

	if len(policyID) > 0 {
		terms["policyID"] = policyID
	}

	if len(status) > 0 {
		terms["status"] = status
	}

	if len(severity) > 0 {
		terms["policySeverity"] = severity
	}

	if len(sourceID) > 0 {
		terms["sourceID"] = sourceID
	}

	if len(resourceTypeID) > 0 {
		terms["resourceType"] = resourceTypeID
	}

	if len(provider) > 0 {
		terms["sourceType"] = provider
	}

	root := map[string]interface{}{}
	root["size"] = 0

	fieldFilter := map[string]interface{}{
		"terms": map[string]interface{}{"field": field, "size": size},
	}
	aggs := map[string]interface{}{
		"field_filter": fieldFilter,
	}
	root["aggs"] = aggs

	boolQuery := make(map[string]interface{})
	if terms != nil && len(terms) > 0 {
		var filters []map[string]interface{}
		for k, vs := range terms {
			filters = append(filters, map[string]interface{}{
				"terms": map[string]interface{}{
					k: vs,
				},
			})
		}

		boolQuery["filter"] = filters
	}
	if len(boolQuery) > 0 {
		root["query"] = map[string]interface{}{
			"bool": boolQuery,
		}
	}

	queryBytes, err := json.Marshal(root)
	if err != nil {
		return nil, err
	}

	var resp AlarmTopFieldResponse
	err = client.Search(context.Background(), summarizer.AlarmIndex, string(queryBytes), &resp)
	return &resp, err
}
