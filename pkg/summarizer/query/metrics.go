package query

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
	summarizer "gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"
)

type FindingMetricsQueryResponse struct {
	Hits FindingMetricsQueryHits `json:"hits"`
}
type FindingMetricsQueryHits struct {
	Total keibi.SearchTotal        `json:"total"`
	Hits  []FindingMetricsQueryHit `json:"hits"`
}
type FindingMetricsQueryHit struct {
	ID      string                    `json:"_id"`
	Score   float64                   `json:"_score"`
	Index   string                    `json:"_index"`
	Type    string                    `json:"_type"`
	Version int64                     `json:"_version,omitempty"`
	Source  summarizer.FindingMetrics `json:"_source"`
	Sort    []interface{}             `json:"sort"`
}

func GetFindingMetrics(client keibi.Client, before, after time.Time) (*summarizer.FindingMetrics, error) {
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"range": map[string]interface{}{
			"described_at": map[string]string{
				"gte": strconv.FormatInt(after.UnixMilli(), 10),
				"lte": strconv.FormatInt(before.UnixMilli(), 10),
			},
		},
	})

	res["size"] = 1
	res["sort"] = []map[string]interface{}{
		{"described_at": "desc"},
	}
	res["query"] = map[string]interface{}{
		"bool": map[string]interface{}{
			"filter": filters,
		},
	}
	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	query := string(b)

	var response FindingMetricsQueryResponse
	err = client.Search(context.Background(), summarizer.MetricsIndex, query, &response)
	if err != nil {
		return nil, err
	}

	for _, hit := range response.Hits.Hits {
		return &hit.Source, nil
	}
	return nil, nil
}
