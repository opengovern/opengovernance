package es

import (
	"context"
	"encoding/json"
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"go.uber.org/zap"
)

type FetchFindingEventsByFindingIDResponse struct {
	Hits struct {
		Hits []struct {
			Source types.FindingEvent `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

func FetchFindingEventsByFindingIDs(logger *zap.Logger, client kaytu.Client, findingID []string) ([]types.FindingEvent, error) {
	request := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"filter": []any{
					map[string]any{
						"terms": map[string][]string{
							"findingEsID": findingID,
						},
					},
				},
			},
		},
		"sort": map[string]any{
			"evaluatedAt": "desc",
		},
		"size": 10000,
	}

	jsonReq, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	logger.Info("Fetching finding events", zap.String("request", string(jsonReq)), zap.String("index", types.FindingEventsIndex))

	var resp FetchFindingEventsByFindingIDResponse
	err = client.Search(context.Background(), types.FindingsIndex, string(jsonReq), &resp)
	if err != nil {
		logger.Error("Failed to fetch finding events", zap.Error(err), zap.String("request", string(jsonReq)), zap.String("index", types.FindingEventsIndex))
		return nil, err
	}
	result := make([]types.FindingEvent, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		result = append(result, hit.Source)
	}
	return result, nil
}
