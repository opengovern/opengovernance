package query

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	summarizer "github.com/kaytu-io/kaytu-engine/pkg/summarizer/es"
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
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
	Sort    []any                       `json:"sort"`
}

func ListBenchmarkSummaries(client keibi.Client, benchmarkID *string) ([]summarizer.BenchmarkSummary, error) {
	var hits []summarizer.BenchmarkSummary
	res := make(map[string]any)
	var filters []any

	if benchmarkID != nil {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"benchmark_id": {*benchmarkID}},
		})
	}

	filters = append(filters, map[string]any{
		"terms": map[string][]string{"report_type": {string(summarizer.BenchmarksSummary)}},
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
		"terms": map[string][]string{"report_type": {string(summarizer.BenchmarksSummaryHistory)}},
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
	err = client.Search(context.Background(), summarizer.BenchmarkSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	for _, hit := range response.Hits.Hits {
		hits = append(hits, hit.Source)
	}
	return hits, nil
}

type LiveBenchmarkAggregatedFindingsQueryResponse struct {
	Aggregations struct {
		PolicyGroup struct {
			Buckets []struct {
				Key         string `json:"key"`
				ResultGroup struct {
					Buckets []struct {
						Key      string `json:"key"`
						DocCount int64  `json:"doc_count"`
					} `json:"buckets"`
				} `json:"result_group"`
			} `json:"buckets"`
		} `json:"policy_group"`
	} `json:"aggregations"`
}

func FetchLiveBenchmarkAggregatedFindings(client keibi.Client, benchmarkID *string, connectionIds []string) (map[string]map[types.ComplianceResult]int, error) {
	var filters []any

	filters = append(filters, map[string]any{
		"term": map[string]bool{"stateActive": true},
	})

	if benchmarkID != nil {
		filters = append(filters, map[string]any{
			"term": map[string]string{"benchmark_id": *benchmarkID},
		})
	}
	if len(connectionIds) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"connectionID": connectionIds},
		})
	}

	queryObj := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"filter": filters,
			},
		},
		"size": 0,
	}
	queryObj["aggs"] = map[string]any{
		"policy_group": map[string]any{
			"terms": map[string]string{"field": "policyID"},
			"aggs": map[string]any{
				"result_group": map[string]any{
					"terms": map[string]string{"field": "result"},
				},
			},
		},
	}

	query, err := json.Marshal(queryObj)
	if err != nil {
		return nil, err
	}

	fmt.Println("query=", string(query), "index=", summarizer.FindingsIndex)

	var response LiveBenchmarkAggregatedFindingsQueryResponse
	err = client.Search(context.Background(), summarizer.FindingsIndex, string(query), &response)
	if err != nil {
		return nil, err
	}

	result := make(map[string]map[types.ComplianceResult]int)
	for _, policyBucket := range response.Aggregations.PolicyGroup.Buckets {
		result[policyBucket.Key] = make(map[types.ComplianceResult]int)
		for _, resultBucket := range policyBucket.ResultGroup.Buckets {
			result[policyBucket.Key][types.ComplianceResult(resultBucket.Key)] = int(resultBucket.DocCount)
		}
	}
	return result, nil
}
