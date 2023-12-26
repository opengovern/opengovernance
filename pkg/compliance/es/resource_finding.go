package es

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"go.uber.org/zap"
)

func DeleteOtherResourceFindingsExcept(logger *zap.Logger, client kaytu.Client, kaytuResourceIDs []string, lessThanJobId uint) error {
	queryMap := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must_not": map[string]any{
					"terms": map[string]any{
						"kaytuResourceID": kaytuResourceIDs,
					},
				},
				"filter": map[string]any{
					"range": map[string]any{
						"jobId": map[string]any{
							"lt": lessThanJobId,
						},
					},
				},
			},
		},
	}
	if len(kaytuResourceIDs) == 0 {
		delete(queryMap["query"].(map[string]any)["bool"].(map[string]any), "must_not")
	}

	query, err := json.Marshal(queryMap)
	if err != nil {
		logger.Error("failed to marshal query", zap.Error(err))
		return err
	}

	es := client.ES()
	_, err = es.DeleteByQuery(
		[]string{types.ResourceFindingsIndex},
		bytes.NewReader(query),
		es.DeleteByQuery.WithContext(context.TODO()),
	)
	if err != nil {
		logger.Error("failed to delete old resource findings", zap.Error(err), zap.String("query", string(query)), zap.Uint("lessThanJobId", lessThanJobId))
		return err
	}

	return nil
}
