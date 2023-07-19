package es

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kaytu-io/kaytu-engine/pkg/describe/es"
	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
)

const (
	EsFetchPageSize       = 10000
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
	Sort    []any             `json:"sort"`
}

func FetchLookupsByResourceIDWildcard(client keibi.Client, resourceID string) (LookupQueryResponse, error) {
	request := make(map[string]any)
	request["size"] = 1
	request["sort"] = []map[string]any{
		{
			"_id": "desc",
		},
	}
	request["query"] = map[string]any{
		"bool": map[string]any{
			"filter": map[string]any{
				"term": map[string]any{
					"resource_id": resourceID,
				},
			},
		},
	}
	b, err := json.Marshal(request)
	if err != nil {
		return LookupQueryResponse{}, err
	}

	fmt.Println("query=", string(b), "index=", InventorySummaryIndex)

	var response LookupQueryResponse
	err = client.Search(context.Background(), InventorySummaryIndex, string(b), &response)
	if err != nil {
		return LookupQueryResponse{}, err
	}
	fmt.Println("response=", response)
	if len(response.Hits.Hits) > 0 {
		return response, nil
	}

	request["query"] = map[string]any{
		"bool": map[string]any{
			"filter": map[string]any{
				"regexp": map[string]any{
					"resource_id": map[string]any{
						"value":            fmt.Sprintf(".*%s.*", resourceID),
						"case_insensitive": true,
					},
				},
			},
		},
	}

	b, err = json.Marshal(request)
	if err != nil {
		return LookupQueryResponse{}, err
	}
	fmt.Println("query=", string(b), "index=", InventorySummaryIndex)
	response = LookupQueryResponse{}
	err = client.Search(context.Background(), InventorySummaryIndex, string(b), &response)
	if err != nil {
		return LookupQueryResponse{}, err
	}
	fmt.Println("response=", response)

	return response, nil
}
