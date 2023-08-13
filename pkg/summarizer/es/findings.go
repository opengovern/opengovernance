package es

import (
	"context"
	"encoding/json"

	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
)

type FetchActiveFindingsResponse struct {
	Hits struct {
		Total kaytu.SearchTotal `json:"total"`
		Hits  []struct {
			ID      string        `json:"_id"`
			Score   float64       `json:"_score"`
			Index   string        `json:"_index"`
			Type    string        `json:"_type"`
			Version int64         `json:"_version,omitempty"`
			Source  types.Finding `json:"_source"`
			Sort    []any         `json:"sort"`
		} `json:"hits"`
	} `json:"hits"`
}

func FetchActiveFindings(client kaytu.Client, searchAfter []any, size int) (FetchActiveFindingsResponse, error) {
	res := make(map[string]any)
	res["query"] = map[string]any{
		"bool": map[string]any{
			"filter": []any{
				map[string]any{
					"term": map[string]string{"stateActive": "true"},
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
		return FetchActiveFindingsResponse{}, err
	}

	var response FetchActiveFindingsResponse
	err = client.Search(context.Background(), types.FindingsIndex, string(b), &response)
	if err != nil {
		return FetchActiveFindingsResponse{}, err
	}

	return response, nil
}
