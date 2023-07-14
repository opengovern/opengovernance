package es

import (
	"context"
	"encoding/json"
	"strconv"

	summarizer "github.com/kaytu-io/kaytu-engine/pkg/summarizer/es"
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
)

type BenchmarkSummaryQueryResponse struct {
	Hits struct {
		Total keibi.SearchTotal `json:"total"`
		Hits  []struct {
			ID      string                 `json:"_id"`
			Score   float64                `json:"_score"`
			Index   string                 `json:"_index"`
			Type    string                 `json:"_type"`
			Version int64                  `json:"_version,omitempty"`
			Source  types.BenchmarkSummary `json:"_source"`
			Sort    []any                  `json:"sort"`
		} `json:"hits"`
	} `json:"hits"`
}

func ListBenchmarkSummaries(client keibi.Client, benchmarkID *string) ([]types.BenchmarkSummary, error) {
	var hits []types.BenchmarkSummary
	res := make(map[string]any)
	var filters []any

	if benchmarkID != nil {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"benchmark_id": {*benchmarkID}},
		})
	}

	filters = append(filters, map[string]any{
		"terms": map[string][]string{"report_type": {string(types.BenchmarksSummary)}},
	})

	sort := []map[string]any{
		{"_id": "desc"},
	}
	res["size"] = summarizer.EsFetchPageSize
	res["sort"] = sort
	res["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}
	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	query := string(b)

	var response BenchmarkSummaryQueryResponse
	err = client.Search(context.Background(), types.BenchmarkSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	for _, hit := range response.Hits.Hits {
		hits = append(hits, hit.Source)
	}
	return hits, nil
}

func FetchBenchmarkSummaryHistory(client keibi.Client, benchmarkID *string, startDate, endDate int64) ([]types.BenchmarkSummary, error) {
	var hits []types.BenchmarkSummary
	res := make(map[string]any)
	var filters []any

	if benchmarkID != nil {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"benchmark_id": {*benchmarkID}},
		})
	}

	filters = append(filters, map[string]any{
		"range": map[string]any{
			"evaluated_at": map[string]string{
				"gte": strconv.FormatInt(startDate, 10),
				"lte": strconv.FormatInt(endDate, 10),
			},
		},
	})

	filters = append(filters, map[string]any{
		"terms": map[string][]string{"report_type": {string(types.BenchmarksSummaryHistory)}},
	})

	sort := []map[string]any{
		{"evaluated_at": "asc", "_id": "desc"},
	}
	res["size"] = summarizer.EsFetchPageSize
	res["sort"] = sort
	res["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}
	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	query := string(b)

	var response BenchmarkSummaryQueryResponse
	err = client.Search(context.Background(), types.BenchmarkSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	for _, hit := range response.Hits.Hits {
		hits = append(hits, hit.Source)
	}
	return hits, nil
}
