package es

import (
	"encoding/json"
	"github.com/opengovern/og-util/pkg/opengovernance-es-sdk"
	"github.com/opengovern/opencomply/pkg/types"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

type ControlViewResponse struct {
	Hits struct {
		Total opengovernance.SearchTotal `json:"total"`
		Hits  []struct {
			ID      string                               `json:"_id"`
			Score   float64                              `json:"_score"`
			Index   string                               `json:"_index"`
			Type    string                               `json:"_type"`
			Version int64                                `json:"_version,omitempty"`
			Source  types.ComplianceQuickScanControlView `json:"_source"`
			Sort    []any                                `json:"sort"`
		}
	}
}

func GetQuickScanControlViewByJobID(ctx context.Context, logger *zap.Logger, client opengovernance.Client, jobID string) (*types.ComplianceQuickScanControlView, error) {
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

	logger.Info("ES Query", zap.String("index", types.ComplianceQuickScanControlViewIndex), zap.String("query", string(b)))

	var response ControlViewResponse
	err = client.Search(ctx, types.ComplianceQuickScanControlViewIndex, string(b), &response)
	if err != nil {
		return nil, err
	}

	if len(response.Hits.Hits) == 0 {
		return nil, nil
	}

	return &response.Hits.Hits[0].Source, nil
}

type ResourceViewResponse struct {
	Hits struct {
		Total opengovernance.SearchTotal `json:"total"`
		Hits  []struct {
			ID      string                                `json:"_id"`
			Score   float64                               `json:"_score"`
			Index   string                                `json:"_index"`
			Type    string                                `json:"_type"`
			Version int64                                 `json:"_version,omitempty"`
			Source  types.ComplianceQuickScanResourceView `json:"_source"`
			Sort    []any                                 `json:"sort"`
		}
	}
}

func GetQuickScanResourceViewByJobID(ctx context.Context, logger *zap.Logger, client opengovernance.Client, jobID string) (*types.ComplianceQuickScanResourceView, error) {
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

	logger.Info("ES Query", zap.String("index", types.ComplianceQuickScanResourceViewIndex), zap.String("query", string(b)))

	var response ResourceViewResponse
	err = client.Search(ctx, types.ComplianceQuickScanResourceViewIndex, string(b), &response)
	if err != nil {
		return nil, err
	}

	if len(response.Hits.Hits) == 0 {
		return nil, nil
	}

	return &response.Hits.Hits[0].Source, nil
}
