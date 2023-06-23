package es

import (
	"context"
	"encoding/json"

	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/es"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
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
	Sort    []any             `json:"sort"`
}

func FetchLookupsByDescribeResourceJobIdList(client keibi.Client, resourceType string, describeResourceJobIdList []uint, searchAfter []any, size int) (LookupQueryResponse, error) {
	res := make(map[string]any)
	var filters []any
	filters = append(filters, map[string]any{
		"terms": map[string][]uint{"resource_job_id": describeResourceJobIdList},
	})

	if searchAfter != nil {
		res["search_after"] = searchAfter
	}

	res["size"] = size
	res["sort"] = []map[string]any{
		{
			"_id": "desc",
		},
	}
	res["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}
	b, err := json.Marshal(res)
	if err != nil {
		return LookupQueryResponse{}, err
	}

	var response LookupQueryResponse
	err = client.Search(context.Background(), es.ResourceTypeToESIndex(resourceType), string(b), &response)
	if err != nil {
		return LookupQueryResponse{}, err
	}

	return response, nil
}

func FetchLookups(client keibi.Client, searchAfter []any, size int) (LookupQueryResponse, error) {
	res := make(map[string]any)
	if searchAfter != nil {
		res["search_after"] = searchAfter
	}

	res["size"] = size
	res["sort"] = []map[string]any{
		{
			"_id": "desc",
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

func FetchLookupByResourceTypes(client keibi.Client, resourceTypes []string, searchAfter []any, size int) (LookupQueryResponse, error) {
	res := make(map[string]any)
	resourceTypes = utils.ToLowerStringSlice(resourceTypes)
	res["query"] = map[string]any{
		"bool": map[string]any{
			"filter": []any{
				map[string]any{
					"terms": map[string][]string{"resource_type": resourceTypes},
				},
			},
		},
	}

	if searchAfter != nil {
		res["search_after"] = searchAfter
	}

	res["size"] = size
	res["sort"] = []map[string]any{
		{
			"_id": "desc",
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
