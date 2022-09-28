package es

import (
	"context"
	"encoding/json"

	es2 "gitlab.com/keibiengine/keibi-engine/pkg/compliance/es"

	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
)

const (
	FindingsIndex = "findings"
)

type FindingQueryResponse struct {
	Hits FindingQueryHits `json:"hits"`
}
type FindingQueryHits struct {
	Total keibi.SearchTotal `json:"total"`
	Hits  []FindingQueryHit `json:"hits"`
}
type FindingQueryHit struct {
	ID      string        `json:"_id"`
	Score   float64       `json:"_score"`
	Index   string        `json:"_index"`
	Type    string        `json:"_type"`
	Version int64         `json:"_version,omitempty"`
	Source  es2.Finding   `json:"_source"`
	Sort    []interface{} `json:"sort"`
}

func FetchFindingsByScheduleJobID(client keibi.Client, scheduleJobID uint, searchAfter []interface{}, size int) (FindingQueryResponse, error) {
	res := make(map[string]interface{})
	var filters []interface{}
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]interface{}{"schedule_job_id": {scheduleJobID}},
	})

	if searchAfter != nil {
		res["search_after"] = searchAfter
	}

	res["size"] = size
	res["sort"] = []map[string]interface{}{
		{
			"_id": "desc",
		},
	}
	res["query"] = map[string]interface{}{
		"bool": map[string]interface{}{
			"filter": filters,
		},
	}
	b, err := json.Marshal(res)
	if err != nil {
		return FindingQueryResponse{}, err
	}

	var response FindingQueryResponse
	err = client.Search(context.Background(), FindingsIndex, string(b), &response)
	if err != nil {
		return FindingQueryResponse{}, err
	}

	return response, nil
}
