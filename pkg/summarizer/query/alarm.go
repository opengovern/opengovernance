package query

import (
	"context"
	"encoding/json"

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
