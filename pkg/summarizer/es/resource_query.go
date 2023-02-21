package es

import (
	"context"
	"encoding/json"

	"gitlab.com/keibiengine/keibi-engine/pkg/describe/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
)

const (
	EsFetchPageSize       = 10000
	EsTermSize            = 10000
	InventorySummaryIndex = "inventory_summary"
)

type LookupQueryResponse struct {
	Hits LookupQueryHits `json:"hits"`
}
type LookupQueryHits struct {
	Total keibi.SearchTotal `json:"total"`
	Hits  []LookupQueryHit  `json:"hits"`
}
type LookupQueryHit struct {
	ID      string            `json:"_id"`
	Score   float64           `json:"_score"`
	Index   string            `json:"_index"`
	Type    string            `json:"_type"`
	Version int64             `json:"_version,omitempty"`
	Source  es.LookupResource `json:"_source"`
	Sort    []interface{}     `json:"sort"`
}

func FetchLookupsByScheduleJobID(client keibi.Client, scheduleJobID uint, searchAfter []interface{}, size int) (LookupQueryResponse, error) {
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
		return LookupQueryResponse{}, err
	}

	var response LookupQueryResponse
	err = client.Search(context.Background(), InventorySummaryIndex, string(b), &response)
	if err != nil {
		return LookupQueryResponse{}, err
	}

	return response, nil
}

func FetchLookupsByDescribeResourceJobIdList(client keibi.Client, describeResourceJobIdList []uint, searchAfter []interface{}, size int) (LookupQueryResponse, error) {
	res := make(map[string]interface{})
	var filters []interface{}
	filters = append(filters, map[string]any{
		"terms": map[string][]uint{"resource_job_id": describeResourceJobIdList},
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
		return LookupQueryResponse{}, err
	}

	var response LookupQueryResponse
	err = client.Search(context.Background(), InventorySummaryIndex, string(b), &response)
	if err != nil {
		return LookupQueryResponse{}, err
	}

	return response, nil
}
