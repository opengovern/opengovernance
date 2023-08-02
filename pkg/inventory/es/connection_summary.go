package es

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/summarizer/es"
	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
)

type ConnectionResourceTypeQueryResponse struct {
	Hits ConnectionResourceTypeQueryHits `json:"hits"`
}
type ConnectionResourceTypeQueryHits struct {
	Total keibi.SearchTotal                `json:"total"`
	Hits  []ConnectionResourceTypeQueryHit `json:"hits"`
}
type ConnectionResourceTypeQueryHit struct {
	ID      string                                `json:"_id"`
	Score   float64                               `json:"_score"`
	Index   string                                `json:"_index"`
	Type    string                                `json:"_type"`
	Version int64                                 `json:"_version,omitempty"`
	Source  es.ConnectionResourceTypeTrendSummary `json:"_source"`
	Sort    []any                                 `json:"sort"`
}

func GetConnectionResourceTypeSummary(client keibi.Client, searchAfter interface{}) (*ConnectionResourceTypeQueryResponse, error) {
	res := make(map[string]any)
	var filters []any

	filters = append(filters, map[string]any{
		"terms": map[string][]string{"report_type": {string(es.ResourceTypeTrendConnectionSummary)}},
	})

	res["size"] = 10000
	if searchAfter != nil {
		res["search_after"] = searchAfter
	}
	res["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}
	res["sort"] = map[string]any{
		"described_at": "desc",
		"_id":          "asc",
	}

	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	query := string(b)
	fmt.Println(query)
	var response ConnectionResourceTypeQueryResponse
	err = client.Search(context.Background(), es.ConnectionSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}
