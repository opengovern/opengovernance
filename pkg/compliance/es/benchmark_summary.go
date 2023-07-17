package es

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"go.uber.org/zap"
)

type ComplianceEvaluationResult struct {
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
					} `json:"buckets"`
				} `json:"benchmark_id_group"`
			} `json:"buckets"`
		} `json:"connection_id_group"`
	} `json:"aggregations"`
}

func FetchBenchmarkSummariesByConnectionIDAtTime(
	logger *zap.Logger, client keibi.Client,
	benchmarkIDs []string, connectors []source.Type, connectionIDs []string, timeAt time.Time) (map[string]ComplianceEvaluationResult, error) {
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
	}
	request["size"] = 0
	request["aggs"] = map[string]any{
		"connection_id_group": map[string]any{
			"terms": map[string]any{
				"field": "connection_id",
				"size":  10000,
			},
			"aggs": map[string]any{
				"benchmark_id_group": map[string]any{
					"terms": map[string]any{
						"field": "benchmark_id",
						"size":  10000,
					},
					"aggs": map[string]any{
						"latest": map[string]any{
							"top_hits": map[string]any{
								"size": 1,
								"_source": []string{
									"benchmark_id",
									"connection_id",
									"connector_types",
									"total_result",
									"total_severity",
									"evaluated_at",
									"report_type",
								},
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

	benchmarkSummaries := make(map[string]ComplianceEvaluationResult)
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
					} `json:"buckets"`
				} `json:"benchmark_id_group"`
			} `json:"buckets"`
		} `json:"connector_group"`
	} `json:"aggregations"`
}

func FetchBenchmarkSummariesByConnectorAtTime(
	logger *zap.Logger, client keibi.Client,
	benchmarkIDs []string, connectors []source.Type, timeAt time.Time) (map[string]ComplianceEvaluationResult, error) {
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
	}
	request["size"] = 0
	request["aggs"] = map[string]any{
		"connector_group": map[string]any{
			"terms": map[string]any{
				"field": "connector_types",
				"size":  10000,
			},
			"aggs": map[string]any{
				"benchmark_id_group": map[string]any{
					"terms": map[string]any{
						"field": "benchmark_id",
						"size":  10000,
					},
					"aggs": map[string]any{
						"latest": map[string]any{
							"top_hits": map[string]any{
								"size": 1,
								"_source": []string{
									"benchmark_id",
									"connection_id",
									"connector_types",
									"total_result",
									"total_severity",
									"evaluated_at",
									"report_type",
								},
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

	benchmarkSummaries := make(map[string]ComplianceEvaluationResult)
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
	benchmarkIDs []string, connectors []source.Type, connectionIDs []string, timeAt time.Time) (map[string]ComplianceEvaluationResult, error) {
	if len(connectionIDs) > 0 {
		return FetchBenchmarkSummariesByConnectionIDAtTime(logger, client, benchmarkIDs, connectors, connectionIDs, timeAt)
	}
	return FetchBenchmarkSummariesByConnectorAtTime(logger, client, benchmarkIDs, connectors, timeAt)
}

type FetchBenchmarkSummaryTrendByConnectionIDResponse struct {
	Aggregations struct {
		BenchmarkIDGroup struct {
			Buckets []struct {
				Key                   string `json:"key"`
				EvaluatedAtRangeGroup struct {
					Buckets []struct {
						From              float64 `json:"from"`
						To                float64 `json:"to"`
						ConnectionIDGroup struct {
							Buckets []struct {
								Key    string `json:"key"`
								Latest struct {
									Hits struct {
										Hits []struct {
											Source types.BenchmarkSummary `json:"_source"`
										} `json:"hits"`
									} `json:"hits"`
								} `json:"latest"`
							} `json:"buckets"`
						} `json:"connection_id_group"`
					} `json:"buckets"`
				} `json:"evaluated_at_range_group"`
			} `json:"buckets"`
		} `json:"benchmark_id_group"`
	} `json:"aggregations"`
}

func FetchBenchmarkSummaryTrendByConnectionID(
	logger *zap.Logger, client keibi.Client,
	benchmarkID []string, connectors []source.Type, connectionID []string, from, to time.Time, datapointCount int) (map[string]map[int]ComplianceEvaluationResult, error) {
	request := make(map[string]any)
	filters := make([]any, 0)
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"evaluated_at": map[string]any{
				"gte": from.Unix(),
				"lte": to.Unix(),
			},
		},
	})
	filters = append(filters, map[string]any{
		"term": map[string]any{
			"report_type": types.BenchmarksSummaryHistory,
		},
	})
	if len(benchmarkID) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{
				"benchmark_id": benchmarkID,
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
	if len(connectionID) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{
				"connection_id": connectionID,
			},
		})
	}
	request["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}
	request["size"] = 0

	startTimeUnix := int(from.Unix())
	endTimeUnix := int(to.Unix())
	step := int(math.Ceil(float64(endTimeUnix-startTimeUnix) / float64(datapointCount)))
	ranges := make([]map[string]any, 0, datapointCount)
	for i := 0; i < datapointCount; i++ {
		ranges = append(ranges, map[string]any{
			"from": float64(startTimeUnix + step*i),
			"to":   float64(startTimeUnix + step*(i+1)),
		})
	}
	request["aggs"] = map[string]any{
		"benchmark_id_group": map[string]any{
			"terms": map[string]any{
				"field": "benchmark_id",
				"size":  10000,
			},
			"aggs": map[string]any{
				"evaluated_at_range_group": map[string]any{
					"range": map[string]any{
						"field":  "evaluated_at",
						"ranges": ranges,
					},
					"aggs": map[string]any{
						"connection_id_group": map[string]any{
							"terms": map[string]any{
								"field": "connection_id",
								"size":  10000,
							},
							"aggs": map[string]any{
								"latest": map[string]any{
									"top_hits": map[string]any{
										"size": 1,
										"sort": map[string]any{
											"evaluated_at": "desc",
										},
									},
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
	logger.Info("FetchBenchmarkSummaryTrendByConnectionID", zap.String("query", string(query)), zap.String("index", types.BenchmarkSummaryIndex))

	var response FetchBenchmarkSummaryTrendByConnectionIDResponse
	if err := client.Search(context.Background(), types.BenchmarkSummaryIndex, string(query), &response); err != nil {
		return nil, err
	}

	result := make(map[string]map[int]ComplianceEvaluationResult)
	for _, benchmarkIDBucket := range response.Aggregations.BenchmarkIDGroup.Buckets {
		if _, ok := result[benchmarkIDBucket.Key]; !ok {
			result[benchmarkIDBucket.Key] = make(map[int]ComplianceEvaluationResult)
		}
		v := result[benchmarkIDBucket.Key]
		for _, rangeGroupBucket := range benchmarkIDBucket.EvaluatedAtRangeGroup.Buckets {
			rangeKey := int(rangeGroupBucket.To)
			evaluation := v[rangeKey]
			for _, evaluatedAtRangeBucket := range rangeGroupBucket.ConnectionIDGroup.Buckets {
				for _, hit := range evaluatedAtRangeBucket.Latest.Hits.Hits {
					evaluation.ComplianceResultSummary.AddComplianceResultSummary(hit.Source.TotalResult)
					evaluation.SeverityResult.AddSeverityResult(hit.Source.TotalSeverity)
				}
			}
			v[rangeKey] = evaluation
		}
		result[benchmarkIDBucket.Key] = v
	}

	return result, nil
}

type FetchBenchmarkSummaryTrendByConnectorResponse struct {
	Aggregations struct {
		BenchmarkIDGroup struct {
			Buckets []struct {
				Key                   string `json:"key"`
				EvaluatedAtRangeGroup struct {
					Buckets []struct {
						From           float64 `json:"from"`
						To             float64 `json:"to"`
						ConnectorGroup struct {
							Buckets []struct {
								Key    string `json:"key"`
								Latest struct {
									Hits struct {
										Hits []struct {
											Source types.BenchmarkSummary `json:"_source"`
										} `json:"hits"`
									} `json:"hits"`
								} `json:"latest"`
							} `json:"buckets"`
						} `json:"connector_group"`
					} `json:"buckets"`
				} `json:"evaluated_at_range_group"`
			} `json:"buckets"`
		} `json:"benchmark_id_group"`
	} `json:"aggregations"`
}

func FetchBenchmarkSummaryTrendByConnector(
	logger *zap.Logger, client keibi.Client,
	benchmarkID []string, connectors []source.Type, from, to time.Time, datapointCount int) (map[string]map[int]ComplianceEvaluationResult, error) {
	request := make(map[string]any)
	filters := make([]any, 0)
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"evaluated_at": map[string]any{
				"gte": from.Unix(),
				"lte": to.Unix(),
			},
		},
	})
	filters = append(filters, map[string]any{
		"term": map[string]any{
			"report_type": types.BenchmarksSummaryHistory,
		},
	})
	if len(benchmarkID) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{
				"benchmark_id": benchmarkID,
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
	}
	request["size"] = 0

	startTimeUnix := int(from.Unix())
	endTimeUnix := int(to.Unix())
	step := int(math.Ceil(float64(endTimeUnix-startTimeUnix) / float64(datapointCount)))
	ranges := make([]map[string]any, 0, datapointCount)
	for i := 0; i < datapointCount; i++ {
		ranges = append(ranges, map[string]any{
			"from": float64(startTimeUnix + step*i),
			"to":   float64(startTimeUnix + step*(i+1)),
		})
	}
	request["aggs"] = map[string]any{
		"benchmark_id_group": map[string]any{
			"terms": map[string]any{
				"field": "benchmark_id",
				"size":  10000,
			},
			"aggs": map[string]any{
				"evaluated_at_range_group": map[string]any{
					"range": map[string]any{
						"field":  "evaluated_at",
						"ranges": ranges,
					},
					"aggs": map[string]any{
						"connector_group": map[string]any{
							"terms": map[string]any{
								"field": "connector_types",
								"size":  10000,
							},
							"aggs": map[string]any{
								"latest": map[string]any{
									"top_hits": map[string]any{
										"size": 1,
										"sort": map[string]any{
											"evaluated_at": "desc",
										},
									},
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
	logger.Info("FetchBenchmarkSummaryTrendByConnectionID", zap.String("query", string(query)), zap.String("index", types.BenchmarkSummaryIndex))

	var response FetchBenchmarkSummaryTrendByConnectorResponse
	if err := client.Search(context.Background(), types.BenchmarkSummaryIndex, string(query), &response); err != nil {
		return nil, err
	}

	result := make(map[string]map[int]ComplianceEvaluationResult)
	for _, benchmarkIDBucket := range response.Aggregations.BenchmarkIDGroup.Buckets {
		if _, ok := result[benchmarkIDBucket.Key]; !ok {
			result[benchmarkIDBucket.Key] = make(map[int]ComplianceEvaluationResult)
		}
		v := result[benchmarkIDBucket.Key]
		for _, rangeGroupBucket := range benchmarkIDBucket.EvaluatedAtRangeGroup.Buckets {
			rangeKey := int(rangeGroupBucket.To)
			evaluation := v[rangeKey]
			for _, evaluatedAtRangeBucket := range rangeGroupBucket.ConnectorGroup.Buckets {
				for _, hit := range evaluatedAtRangeBucket.Latest.Hits.Hits {
					evaluation.ComplianceResultSummary.AddComplianceResultSummary(hit.Source.TotalResult)
					evaluation.SeverityResult.AddSeverityResult(hit.Source.TotalSeverity)
				}
			}
			v[rangeKey] = evaluation
		}
		result[benchmarkIDBucket.Key] = v
	}

	return result, nil
}

func FetchBenchmarkSummaryTrend(
	logger *zap.Logger, client keibi.Client,
	benchmarkID []string, connectors []source.Type, connectionID []string, from, to time.Time, datapointCount int) (map[string]map[int]ComplianceEvaluationResult, error) {
	if len(connectionID) > 0 {
		return FetchBenchmarkSummaryTrendByConnectionID(logger, client, benchmarkID, connectors, connectionID, from, to, datapointCount)
	}
	return FetchBenchmarkSummaryTrendByConnector(logger, client, benchmarkID, connectors, from, to, datapointCount)
}
