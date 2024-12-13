package es

import (
	"bytes"
	"encoding/json"
	"github.com/opengovern/og-util/pkg/opengovernance-es-sdk"
	"github.com/opengovern/opencomply/pkg/types"
	"go.uber.org/zap"
)

func CleanupSummariesForJobs(logger *zap.Logger, es opengovernance.Client, jobIds []uint) error {
	root := make(map[string]any)
	root["query"] = map[string]any{
		"bool": map[string]any{
			"filter": []any{
				map[string]any{
					"terms": map[string]any{
						"JobID": jobIds,
					},
				},
			},
		},
	}
	query, err := json.Marshal(root)
	if err != nil {
		return err
	}

	logger.Info("Query to delete summaries", zap.ByteString("query", query))
	res, err := es.ES().DeleteByQuery([]string{types.BenchmarkSummaryIndex}, bytes.NewReader(query))
	if err != nil {
		return err
	}

	logger.Info("Delete summaries response", zap.String("resp", res.String()))

	opengovernance.CloseSafe(res)
	return nil
}
