package es

import (
	"context"
	"encoding/json"
	"fmt"
	types2 "github.com/kaytu-io/kaytu-engine/pkg/compliance/summarizer/types"
	"strings"
	"time"

	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"go.uber.org/zap"
)

type TrendDatapoint struct {
	DateEpoch int64
	Score     float64
}

type FetchBenchmarkSummaryTrendResponse struct {
	Hits struct {
		Hits []struct {
			Fields map[string][]any `json:"fields"`
		} `json:"hits"`
	} `json:"hits"`
}

func FetchBenchmarkSummaryTrend(logger *zap.Logger, client kaytu.Client, benchmarkIDs, connectionIDs, resourceCollections []string, from, to time.Time) (map[string][]TrendDatapoint, error) {
	idx := types2.BenchmarkSummaryIndex

	includes := []string{"BenchmarkID", "BenchmarkResult.SecurityScore", "EvaluatedAtEpoch"}
	for _, connectionID := range connectionIDs {
		includes = append(includes, fmt.Sprintf("Connections.%s.SecurityScore", connectionID))
	}
	for _, resourceCollection := range resourceCollections {
		includes = append(includes, fmt.Sprintf("ResourceCollections.%s.SecurityScore", resourceCollection))
	}

	request := map[string]any{
		"_source": false,
		"fields":  includes,
	}

	filters := make([]any, 0)
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"EvaluatedAtEpoch": map[string]any{
				"lte": to.Unix(),
				"gte": from.Unix(),
			},
		},
	})
	if len(benchmarkIDs) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{
				"BenchmarkID": benchmarkIDs,
			},
		})
	}

	request["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}

	query, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	logger.Info("FetchBenchmarkSummariesByConnectionIDAtTime", zap.String("query", string(query)), zap.String("index", idx))

	var response FetchBenchmarkSummaryTrendResponse
	err = client.Search(context.Background(), idx, string(query), &response)
	if err != nil {
		return nil, err
	}

	trend := make(map[string][]TrendDatapoint)

	for _, summary := range response.Hits.Hits {
		var date int64
		var sum float64
		var benchmarkID string
		for k, v := range summary.Fields {
			if len(v) != 1 {
				return nil, fmt.Errorf("invalid length %d", len(v))
			}
			if k == "EvaluatedAtEpoch" {
				date = int64(v[0].(float64))
			} else if strings.HasSuffix(k, "SecurityScore") {
				sum += v[0].(float64)
			} else if k == "BenchmarkID" {
				benchmarkID = v[0].(string)
			}
		}
		trend[benchmarkID] = append(trend[benchmarkID], TrendDatapoint{
			DateEpoch: date,
			Score:     sum,
		})
	}
	return trend, nil
}

type ListBenchmarkSummariesAtTimeResponse struct {
	Aggregations struct {
		Summaries struct {
			Buckets []struct {
				Key        string `json:"key"`
				DocCount   int    `json:"doc_count"`
				LastResult struct {
					Hits struct {
						Hits []struct {
							Source types2.BenchmarkSummary `json:"_source"`
						} `json:"hits"`
					} `json:"hits"`
				} `json:"last_result"`
			} `json:"buckets"`
		} `json:"summaries"`
	} `json:"aggregations"`
}

func ListBenchmarkSummariesAtTime(logger *zap.Logger, client kaytu.Client,
	benchmarkIDs []string, connectionIDs []string, resourceCollections []string,
	timeAt time.Time) (map[string]types2.BenchmarkSummary, error) {

	idx := types2.BenchmarkSummaryIndex

	includes := []string{"BenchmarkResult", "EvaluatedAtEpoch"}
	for _, connectionID := range connectionIDs {
		includes = append(includes, fmt.Sprintf("Connections.%s", connectionID))
	}
	for _, resourceCollection := range resourceCollections {
		includes = append(includes, fmt.Sprintf("ResourceCollections.%s", resourceCollection))
	}

	request := map[string]any{
		"aggs": map[string]any{
			"summaries": map[string]any{
				"terms": map[string]any{
					"field": "BenchmarkID",
				},
				"aggs": map[string]any{
					"last_result": map[string]any{
						"top_hits": map[string]any{
							"sort": []map[string]any{
								{
									"JobID": "desc",
								},
							},
							"_source": map[string]any{
								"includes": includes,
							},
							"size": 1,
						},
					},
				},
			},
		},
		"size": 0,
	}

	filters := make([]any, 0)
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"EvaluatedAtEpoch": map[string]any{
				"lte": timeAt.Unix(),
			},
		},
	})
	if len(benchmarkIDs) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{
				"BenchmarkID": benchmarkIDs,
			},
		})
	}

	request["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}

	query, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	logger.Info("FetchBenchmarkSummariesByConnectionIDAtTime", zap.String("query", string(query)), zap.String("index", idx))

	var response ListBenchmarkSummariesAtTimeResponse
	err = client.Search(context.Background(), idx, string(query), &response)
	if err != nil {
		return nil, err
	}

	benchmarkSummaries := make(map[string]types2.BenchmarkSummary)
	for _, summary := range response.Aggregations.Summaries.Buckets {
		for _, hit := range summary.LastResult.Hits.Hits {
			benchmarkSummaries[summary.Key] = hit.Source
		}
	}
	return benchmarkSummaries, nil
}

type BenchmarkConnectionSummaryResponse struct {
	Aggregations struct {
		LastResult struct {
			Hits struct {
				Hits []struct {
					Source types2.BenchmarkSummary `json:"_source"`
				} `json:"hits"`
			} `json:"hits"`
		} `json:"last_result"`
	} `json:"aggregations"`
}

func BenchmarkConnectionSummary(logger *zap.Logger, client kaytu.Client, benchmarkID string) (map[string]types2.Result, int64, error) {
	includes := []string{"Connections", "EvaluatedAtEpoch"}
	request := map[string]any{
		"aggs": map[string]any{
			"last_result": map[string]any{
				"top_hits": map[string]any{
					"sort": []map[string]any{
						{
							"JobID": "desc",
						},
					},
					"_source": map[string]any{
						"includes": includes,
					},
					"size": 1,
				},
			},
		},
		"query": map[string]any{
			"bool": map[string]any{
				"filter": []map[string]any{
					{
						"term": map[string]any{
							"BenchmarkID": benchmarkID,
						},
					},
				},
			},
		},
		"size": 0,
	}

	queryBytes, err := json.Marshal(request)
	if err != nil {
		return nil, -1, err
	}

	logger.Info("BenchmarkConnectionSummary", zap.String("query", string(queryBytes)))
	var resp BenchmarkConnectionSummaryResponse
	err = client.Search(context.Background(), types2.BenchmarkSummaryIndex, string(queryBytes), &resp)
	if err != nil {
		return nil, -1, err
	}

	for _, res := range resp.Aggregations.LastResult.Hits.Hits {
		return res.Source.Connections.Connections, res.Source.EvaluatedAtEpoch, nil
	}
	return nil, 0, nil
}

func BenchmarkPolicySummary(logger *zap.Logger, client kaytu.Client, benchmarkID string) (map[string]types2.PolicyResult, int64, error) {
	includes := []string{"Policies", "EvaluatedAtEpoch"}
	request := map[string]any{
		"aggs": map[string]any{
			"last_result": map[string]any{
				"top_hits": map[string]any{
					"sort": []map[string]any{
						{
							"JobID": "desc",
						},
					},
					"_source": map[string]any{
						"includes": includes,
					},
					"size": 1,
				},
			},
		},
		"query": map[string]any{
			"bool": map[string]any{
				"filter": []map[string]any{
					{
						"term": map[string]any{
							"BenchmarkID": benchmarkID,
						},
					},
				},
			},
		},
		"size": 0,
	}

	queryBytes, err := json.Marshal(request)
	if err != nil {
		return nil, -1, err
	}

	logger.Info("BenchmarkPolicySummary", zap.String("query", string(queryBytes)))
	var resp BenchmarkConnectionSummaryResponse
	err = client.Search(context.Background(), types2.BenchmarkSummaryIndex, string(queryBytes), &resp)
	if err != nil {
		return nil, -1, err
	}

	for _, res := range resp.Aggregations.LastResult.Hits.Hits {
		return res.Source.Connections.Policies, res.Source.EvaluatedAtEpoch, nil
	}
	return nil, 0, nil
}
