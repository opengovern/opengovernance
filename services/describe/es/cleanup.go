package es

import (
	"bytes"
	"encoding/json"
	"github.com/opengovern/og-util/pkg/opengovernance-es-sdk"
	"github.com/opengovern/opencomply/pkg/types"
)

func CleanupSummariesForJobs(es opengovernance.Client, jobIds []uint) error {
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

	res, err := es.ES().DeleteByQuery([]string{types.BenchmarkSummaryIndex}, bytes.NewReader(query))
	if err != nil {
		return err
	}

	opengovernance.CloseSafe(res)
	return nil
}
