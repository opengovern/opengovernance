package es

import (
	"encoding/json"
	"github.com/opengovern/og-util/pkg/opengovernance-es-sdk"
	"github.com/opengovern/opencomply/pkg/types"
	"golang.org/x/net/context"
)

type AuditSummaryResponse struct {
	Hits struct {
		Total opengovernance.SearchTotal `json:"total"`
		Hits  []struct {
			ID      string             `json:"_id"`
			Score   float64            `json:"_score"`
			Index   string             `json:"_index"`
			Type    string             `json:"_type"`
			Version int64              `json:"_version,omitempty"`
			Source  types.AuditSummary `json:"_source"`
			Sort    []any              `json:"sort"`
		}
	}
}

func GetAuditSummaryByJobID(ctx context.Context, client opengovernance.Client, jobID string) (*types.AuditSummary, error) {
	request := make(map[string]any)
	request["size"] = 1
	request["query"] = map[string]any{
		"bool": map[string]any{
			"filter": map[string]any{
				"term": map[string]any{
					"job_summary.job_id": jobID,
				},
			},
		},
	}
	b, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	var response AuditSummaryResponse
	err = client.Search(ctx, types.AuditSummaryIndex, string(b), &response)
	if err != nil {
		return nil, err
	}

	if len(response.Hits.Hits) == 0 {
		return nil, nil
	}

	return &response.Hits.Hits[0].Source, nil
}
