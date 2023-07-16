package es

import (
	"context"
	"encoding/json"
	"time"

	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"go.uber.org/zap"
)

type ComplianceSeverityResult struct {
	ComplianceResultSummary types.ComplianceResultSummary
	SeverityResult          types.SeverityResult
}

type FetchBenchmarkSummariesByConnectionIDAtTimeResponse struct {
	Aggregations struct {
		ConnectionIDGroup struct {
			Buckets []struct {
				Key              string `json:"key"`
				BenchmarkIDGroup struct {
					Buckets []struct {
						Key    string `json:"key"`
						Latest struct {
							Hits struct {
								Hits []struct {
									Source types.BenchmarkSummary `json:"_source"`
								} `json:"hits"`
							} `json:"hits"`
						} `json:"latest"`
					} `json:"benchmark_id_group"`
				} `json:"benchmark_id_group"`
			} `json:"buckets"`
		} `json:"connection_id_group"`
	} `json:"aggregations"`
}

func FetchBenchmarkSummariesByConnectionIDAtTime(
	logger *zap.Logger, client keibi.Client,
	benchmarkIDs []string, connectors []source.Type, connectionIDs []string, timeAt time.Time) (map[string]ComplianceSeverityResult, error) {
	request := make(map[string]any)
	filters := make([]any, 0)
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"evaluated_at": map[string]any{
				"lte": timeAt.Unix(),
			},
		},
	})
	filters = append(filters, map[string]any{
		"term": map[string]any{
			"report_type": types.BenchmarksSummaryHistory,
		},
	})
	if len(benchmarkIDs) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{
				"benchmark_id": benchmarkIDs,
			},
		})
	}
	if len(connectors) > 0 {
		connectorsStr := make([]string, 0, len(connectors))
		for _, connector := range connectors {
			connectorsStr = append(connectorsStr, connector.String())
		}
		filters = append(filters, map[string]any{
			"terms": map[string][]string{
				"connector_types": connectorsStr,
			},
		})
	}
	if len(connectionIDs) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{
				"connection_id": connectionIDs,
			},
		})
	}

	request["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
		"size": 0,
	}
	request["aggs"] = map[string]any{
		"connection_id_group": map[string]any{
			"terms": map[string]string{
				"field": "connection_id",
			},
			"aggs": map[string]any{
				"benchmark_id_group": map[string]any{
					"terms": map[string]string{
						"field": "benchmark_id",
					},
					"aggs": map[string]any{
						"latest": map[string]any{
							"top_hits": map[string]any{
								"size": 1,
								"sort": map[string]string{
									"evaluated_at": "desc",
								},
							},
						},
					},
				},
			},
		},
	}

	query, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	logger.Info("FetchBenchmarkSummariesByConnectionIDAtTime", zap.String("query", string(query)), zap.String("index", types.BenchmarkSummaryIndex))

	var response FetchBenchmarkSummariesByConnectionIDAtTimeResponse
	err = client.Search(context.Background(), types.BenchmarkSummaryIndex, string(query), &response)
	if err != nil {
		return nil, err
	}

	benchmarkSummaries := make(map[string]ComplianceSeverityResult)
	for _, connectionIDBucket := range response.Aggregations.ConnectionIDGroup.Buckets {
		for _, benchmarkIDBucket := range connectionIDBucket.BenchmarkIDGroup.Buckets {
			v := benchmarkSummaries[benchmarkIDBucket.Key]
			for _, hit := range benchmarkIDBucket.Latest.Hits.Hits {
				v.ComplianceResultSummary.AddComplianceResultSummary(hit.Source.TotalResult)
				v.SeverityResult.AddSeverityResult(hit.Source.TotalSeverity)
			}
			benchmarkSummaries[benchmarkIDBucket.Key] = v
		}
	}

	return benchmarkSummaries, nil
}

type FetchBenchmarkSummariesByConnectorAtTimeResponse struct {
	Aggregations struct {
		ConnectorGroup struct {
			Buckets []struct {
				Key              string `json:"key"`
				BenchmarkIDGroup struct {
					Buckets []struct {
						Key    string `json:"key"`
						Latest struct {
							Hits struct {
								Hits []struct {
									Source types.BenchmarkSummary `json:"_source"`
								} `json:"hits"`
							} `json:"hits"`
						} `json:"latest"`
					} `json:"benchmark_id_group"`
				} `json:"benchmark_id_group"`
			} `json:"buckets"`
		} `json:"connector_group"`
	} `json:"aggregations"`
}

func FetchBenchmarkSummariesByConnectorAtTime(
	logger *zap.Logger, client keibi.Client,
	benchmarkIDs []string, connectors []source.Type, timeAt time.Time) (map[string]ComplianceSeverityResult, error) {
	request := make(map[string]any)
	filters := make([]any, 0)
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"evaluated_at": map[string]any{
				"lte": timeAt.Unix(),
			},
		},
	})
	filters = append(filters, map[string]any{
		"term": map[string]any{
			"report_type": types.BenchmarksConnectorSummaryHistory,
		},
	})
	if len(benchmarkIDs) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{
				"benchmark_id": benchmarkIDs,
			},
		})
	}
	if len(connectors) > 0 {
		connectorsStr := make([]string, 0, len(connectors))
		for _, connector := range connectors {
			connectorsStr = append(connectorsStr, connector.String())
		}
		filters = append(filters, map[string]any{
			"terms": map[string][]string{
				"connector_types": connectorsStr,
			},
		})
	}

	request["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
		"size": 0,
	}
	request["aggs"] = map[string]any{
		"connector_group": map[string]any{
			"terms": map[string]string{
				"field": "connector_types",
			},
			"aggs": map[string]any{
				"benchmark_id_group": map[string]any{
					"terms": map[string]string{
						"field": "benchmark_id",
					},
					"aggs": map[string]any{
						"latest": map[string]any{
							"top_hits": map[string]any{
								"size": 1,
								"sort": map[string]string{
									"evaluated_at": "desc",
								},
							},
						},
					},
				},
			},
		},
	}

	query, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	logger.Info("FetchBenchmarkSummariesByConnectionIDAtTime", zap.String("query", string(query)), zap.String("index", types.BenchmarkSummaryIndex))

	var response FetchBenchmarkSummariesByConnectorAtTimeResponse
	err = client.Search(context.Background(), types.BenchmarkSummaryIndex, string(query), &response)
	if err != nil {
		return nil, err
	}

	benchmarkSummaries := make(map[string]ComplianceSeverityResult)
	for _, connectorBucket := range response.Aggregations.ConnectorGroup.Buckets {
		for _, benchmarkIDBucket := range connectorBucket.BenchmarkIDGroup.Buckets {
			v := benchmarkSummaries[benchmarkIDBucket.Key]
			for _, hit := range benchmarkIDBucket.Latest.Hits.Hits {
				v.ComplianceResultSummary.AddComplianceResultSummary(hit.Source.TotalResult)
				v.SeverityResult.AddSeverityResult(hit.Source.TotalSeverity)
			}
			benchmarkSummaries[benchmarkIDBucket.Key] = v
		}
	}

	return benchmarkSummaries, nil
}

func FetchBenchmarkSummariesAtTime(
	logger *zap.Logger, client keibi.Client,
	benchmarkIDs []string, connectors []source.Type, connectionIDs []string, timeAt time.Time) (map[string]ComplianceSeverityResult, error) {
	if len(connectionIDs) > 0 {
		return FetchBenchmarkSummariesByConnectionIDAtTime(logger, client, benchmarkIDs, connectors, connectionIDs, timeAt)
	}
	return FetchBenchmarkSummariesByConnectorAtTime(logger, client, benchmarkIDs, connectors, timeAt)
}
