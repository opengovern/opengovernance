package query

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
	summarizer "gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"
)

type BenchmarkSummaryQueryResponse struct {
	Hits BenchmarkSummaryQueryHits `json:"hits"`
}
type BenchmarkSummaryQueryHits struct {
	Total keibi.SearchTotal          `json:"total"`
	Hits  []BenchmarkSummaryQueryHit `json:"hits"`
}
type BenchmarkSummaryQueryHit struct {
	ID      string                      `json:"_id"`
	Score   float64                     `json:"_score"`
	Index   string                      `json:"_index"`
	Type    string                      `json:"_type"`
	Version int64                       `json:"_version,omitempty"`
	Source  summarizer.BenchmarkSummary `json:"_source"`
	Sort    []interface{}               `json:"sort"`
}

func ListBenchmarkSummaries(client keibi.Client, benchmarkID *string) ([]summarizer.BenchmarkSummary, error) {
	var hits []summarizer.BenchmarkSummary
	res := make(map[string]interface{})
	var filters []interface{}

	if benchmarkID != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"benchmark_id": {*benchmarkID}},
		})
	}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"report_type": {string(summarizer.BenchmarksSummary)}},
	})

	sort := []map[string]interface{}{
		{"_id": "desc"},
	}
	res["size"] = summarizer.EsFetchPageSize
	res["sort"] = sort
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

	var response BenchmarkSummaryQueryResponse
	err = client.Search(context.Background(), summarizer.BenchmarkSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	for _, hit := range response.Hits.Hits {
		hits = append(hits, hit.Source)
	}
	return hits, nil
}

func FetchBenchmarkSummaryHistory(client keibi.Client, benchmarkID *string, startDate, endDate int64) ([]summarizer.BenchmarkSummary, error) {
	var hits []summarizer.BenchmarkSummary
	res := make(map[string]interface{})
	var filters []interface{}

	if benchmarkID != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"benchmark_id": {*benchmarkID}},
		})
	}

	filters = append(filters, map[string]interface{}{
		"range": map[string]interface{}{
			"evaluated_at": map[string]string{
				"gte": strconv.FormatInt(startDate, 10),
				"lte": strconv.FormatInt(endDate, 10),
			},
		},
	})

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"report_type": {string(summarizer.BenchmarksSummaryHistory)}},
	})

	sort := []map[string]interface{}{
		{"evaluated_at": "asc", "_id": "desc"},
	}
	res["size"] = summarizer.EsFetchPageSize
	res["sort"] = sort
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

	var response BenchmarkSummaryQueryResponse
	err = client.Search(context.Background(), summarizer.BenchmarkSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	for _, hit := range response.Hits.Hits {
		hits = append(hits, hit.Source)
	}
	return hits, nil
}
