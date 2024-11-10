package es

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/opengovern/og-util/pkg/opengovernance-es-sdk"
	"github.com/opengovern/og-util/pkg/source"
	"github.com/opengovern/opengovernance/pkg/compliance/api"
	"github.com/opengovern/opengovernance/pkg/types"
	"github.com/opengovern/opengovernance/pkg/utils"
	"go.uber.org/zap"
	"strings"
	"time"
)

type ResourceFindingsQueryHit struct {
	ID      string                `json:"_id"`
	Score   float64               `json:"_score"`
	Index   string                `json:"_index"`
	Type    string                `json:"_type"`
	Version int64                 `json:"_version,omitempty"`
	Source  types.ResourceFinding `json:"_source"`
	Sort    []any                 `json:"sort"`
}

type ResourceFindingsQueryResponse struct {
	Hits struct {
		Total opengovernance.SearchTotal `json:"total"`
		Hits  []ResourceFindingsQueryHit `json:"hits"`
	} `json:"hits"`
	PitID string `json:"pit_id"`
}

type ResourceFindingPaginator struct {
	paginator *opengovernance.BaseESPaginator
}

func NewResourceFindingPaginator(client opengovernance.Client, idx string, filters []opengovernance.BoolFilter, limit *int64, sort []map[string]any) (ResourceFindingPaginator, error) {
	paginator, err := opengovernance.NewPaginatorWithSort(client.ES(), idx, filters, limit, sort)
	if err != nil {
		return ResourceFindingPaginator{}, err
	}

	p := ResourceFindingPaginator{
		paginator: paginator,
	}

	return p, nil
}

func (p ResourceFindingPaginator) HasNext() bool {
	return !p.paginator.Done()
}

func (p ResourceFindingPaginator) Close(ctx context.Context) error {
	return p.paginator.Deallocate(ctx)
}

func (p ResourceFindingPaginator) NextPage(ctx context.Context) ([]types.ResourceFinding, error) {
	var response ResourceFindingsQueryResponse
	err := p.paginator.SearchWithLog(ctx, &response, true)
	if err != nil {
		return nil, err
	}

	var values []types.ResourceFinding
	for _, hit := range response.Hits.Hits {
		values = append(values, hit.Source)
	}

	hits := int64(len(response.Hits.Hits))
	if hits > 0 {
		p.paginator.UpdateState(hits, response.Hits.Hits[hits-1].Sort, response.PitID)
	} else {
		p.paginator.UpdateState(hits, nil, "")
	}

	return values, nil
}

func ResourceFindingsQuery(ctx context.Context, logger *zap.Logger, client opengovernance.Client, connector []source.Type, connectionID []string,
	notConnectionID []string, resourceCollection []string, resourceTypes []string, benchmarkID []string, controlID []string,
	severity []types.ComplianceResultSeverity, evaluatedAtFrom *time.Time, evaluatedAtTo *time.Time, conformanceStatuses []types.ConformanceStatus,
	sorts []api.ResourceFindingsSort, pageSizeLimit int, searchAfter []any, summaryJobIDs []string) ([]ResourceFindingsQueryHit, int64, error) {

	nestedFilters := make([]map[string]any, 0)
	if len(connector) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string]any{
				"complianceResults.connector": connector,
			},
		})
	}
	if len(connectionID) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string]any{
				"complianceResults.connectionID": connectionID,
			},
		})
	}
	if len(notConnectionID) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"bool": map[string]any{
				"must_not": []map[string]any{
					{
						"terms": map[string]any{
							"complianceResults.connectionID": notConnectionID,
						},
					},
				},
			},
		})
	}
	if len(benchmarkID) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string]any{
				"complianceResults.benchmarkID": benchmarkID,
			},
		})
	}
	if len(controlID) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string]any{
				"complianceResults.controlID": controlID,
			},
		})
	}
	if len(severity) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string]any{
				"complianceResults.severity": severity,
			},
		})
	}
	if len(conformanceStatuses) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string]any{
				"complianceResults.conformanceStatus": conformanceStatuses,
			},
		})
	}

	filters := make([]map[string]any, 0)
	if len(resourceTypes) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string]any{
				"resourceType": utils.ToLowerStringSlice(resourceTypes),
			},
		})
	}
	if len(resourceCollection) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string]any{
				"resourceCollection": resourceCollection,
			},
		})
	}
	if evaluatedAtFrom != nil && evaluatedAtTo != nil {
		filters = append(filters, map[string]any{
			"range": map[string]any{
				"evaluatedAt": map[string]any{
					"gte": fmt.Sprintf("%d", evaluatedAtFrom.UnixMilli()),
					"lte": fmt.Sprintf("%d", evaluatedAtTo.UnixMilli()),
				},
			},
		})
	} else if evaluatedAtFrom != nil {
		filters = append(filters, map[string]any{
			"range": map[string]any{
				"evaluatedAt": map[string]any{
					"gte": fmt.Sprintf("%d", evaluatedAtFrom.UnixMilli()),
				},
			},
		})
	} else if evaluatedAtTo != nil {
		filters = append(filters, map[string]any{
			"range": map[string]any{
				"evaluatedAt": map[string]any{
					"lte": fmt.Sprintf("%d", evaluatedAtTo.UnixMilli()),
				},
			},
		})
	}

	if len(nestedFilters) > 0 {
		filters = append(filters, map[string]any{"nested": map[string]any{
			"path": "complianceResults",
			"query": map[string]any{
				"bool": map[string]any{
					"filter": nestedFilters,
				},
			},
		}})
	}

	requestMap := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"filter": filters,
			},
		},
		"size": pageSizeLimit,
	}
	if len(filters) == 0 {
		delete(requestMap, "query")
	}
	if len(searchAfter) > 0 {
		requestMap["search_after"] = searchAfter
	}

	requestSort := make([]map[string]any, 0, len(sorts)+1)
	for _, sort := range sorts {
		switch {
		case sort.OpenGovernanceResourceID != nil:
			requestSort = append(requestSort, map[string]any{
				"opengovernanceResourceID": *sort.OpenGovernanceResourceID,
			})
		case sort.ResourceType != nil:
			requestSort = append(requestSort, map[string]any{
				"resourceType": *sort.ResourceType,
			})
		case sort.ResourceName != nil:
			requestSort = append(requestSort, map[string]any{
				"resourceName": *sort.ResourceName,
			})
		case sort.ResourceLocation != nil:
			requestSort = append(requestSort, map[string]any{
				"resourceLocation": *sort.ResourceLocation,
			})
		case sort.ConformanceStatus != nil:
			requestSort = append(requestSort, map[string]any{
				"complianceResults.conformanceStatus": *sort.ConformanceStatus,
			})
		case sort.FailedCount != nil:
			scriptSource :=
				fmt.Sprintf(`int total = 0; 
for (int i=0; i<params['_source']['complianceResults'].length;++i) { 
  if(params['_source']['complianceResults'][i]['conformanceStatus'] != '%s' && params['_source']['complianceResults'][i]['conformanceStatus'] != '%s' && params['_source']['complianceResults'][i]['conformanceStatus'] != '%s') 
    total+=1;
  } 
return total;`, types.ConformanceStatusOK, types.ConformanceStatusINFO, types.ConformanceStatusSKIP)
			requestSort = append(requestSort, map[string]any{
				"_script": map[string]any{
					"type": "number",
					"script": map[string]any{
						"lang":   "painless",
						"source": scriptSource,
					},
					"order": *sort.FailedCount,
				},
			})
		}
	}
	requestSort = append(requestSort, map[string]any{
		"_id": "asc",
	})
	requestMap["sort"] = requestSort

	request, err := json.Marshal(requestMap)
	if err != nil {
		logger.Error("resourceFindingsQuery - failed to marshal request", zap.Error(err), zap.Any("request", requestMap))
		return nil, 0, err
	}
	logger.Info("ResourceFindingsQuery", zap.String("request", string(request)), zap.String("index", types.ResourceFindingsIndex))

	var response ResourceFindingsQueryResponse
	err = client.SearchWithTrackTotalHits(ctx, types.ResourceFindingsIndex, string(request), nil, &response, true)
	if err != nil {
		return nil, 0, err
	}

	return response.Hits.Hits, response.Hits.Total.Value, nil
}

type GetPerBenchmarkResourceSeverityResultResponse struct {
	Aggregations struct {
		ComplianceResults struct {
			ConformanceFilter struct {
				BenchmarkGroup struct {
					Buckets []struct {
						Key           string `json:"key"`
						SeverityGroup struct {
							Buckets []struct {
								Key           string `json:"key"`
								ResourceCount struct {
									DocCount int `json:"doc_count"`
								} `json:"resourceCount"`
							} `json:"buckets"`
						} `json:"severityGroup"`
					} `json:"buckets"`
				} `json:"benchmarkGroup"`
			} `json:"conformanceFilter"`
		} `json:"complianceResults"`
	} `json:"aggregations"`
}

func GetPerBenchmarkResourceSeverityResult(ctx context.Context, logger *zap.Logger, client opengovernance.Client,
	benchmarkIDs []string, connectionIDs []string, resourceCollections []string,
	severities []types.ComplianceResultSeverity, conformanceStatuses []types.ConformanceStatus) (map[string]types.SeverityResultWithTotal, error) {
	request := make(map[string]any)
	filters := make([]map[string]any, 0)
	nestedFilters := make([]map[string]any, 0)
	if len(benchmarkIDs) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string][]string{
				"complianceResults.benchmarkID": benchmarkIDs,
			},
		})
	}
	if len(connectionIDs) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string][]string{
				"complianceResults.connectionID": connectionIDs,
			},
		})
	}
	if len(resourceCollections) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{
				"resourceCollections": resourceCollections,
			},
		})
	}
	if len(severities) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string]any{
				"complianceResults.severity": severities,
			},
		})
	}
	if len(conformanceStatuses) == 0 {
		conformanceStatuses = types.GetConformanceStatuses()
	}
	nestedFilters = append(nestedFilters, map[string]any{
		"terms": map[string]any{
			"complianceResults.conformanceStatus": conformanceStatuses,
		},
	})

	requestQuery := make(map[string]any, 0)
	if len(nestedFilters) > 0 {
		filters = append(filters, map[string]any{
			"nested": map[string]any{
				"path":  "complianceResults",
				"query": map[string]any{"bool": map[string]any{"filter": nestedFilters}},
			},
		})
	}
	if len(filters) > 0 {
		requestQuery["bool"] = map[string]any{
			"filter": filters,
		}
	}
	if len(requestQuery) > 0 {
		request["query"] = requestQuery
	}
	request["size"] = 0

	request["aggs"] = map[string]any{
		"complianceResults": map[string]any{
			"nested": map[string]any{
				"path": "complianceResults",
			},
			"aggs": map[string]any{
				"conformanceFilter": map[string]any{
					"filter": map[string]any{
						"terms": map[string]any{
							"complianceResults.conformanceStatus": conformanceStatuses,
						},
					},
					"aggs": map[string]any{
						"benchmarkGroup": map[string]any{
							"terms": map[string]any{
								"field": "complianceResults.benchmarkID",
								"size":  10000,
							},
							"aggs": map[string]any{
								"severityGroup": map[string]any{
									"terms": map[string]any{
										"field": "complianceResults.severity",
										"size":  10000,
									},
									"aggs": map[string]any{
										"resourceCount": map[string]any{
											"reverse_nested": map[string]any{},
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
		logger.Error("GetPerBenchmarkResourceSeverityResult", zap.Error(err), zap.Any("request", request))
	}

	logger.Info("GetPerBenchmarkResourceSeverityResult", zap.String("query", string(query)), zap.String("index", types.ResourceFindingsIndex))
	var response GetPerBenchmarkResourceSeverityResultResponse
	err = client.Search(ctx, types.ResourceFindingsIndex, string(query), &response)
	if err != nil {
		logger.Error("GetPerBenchmarkResourceSeverityResult", zap.Error(err), zap.String("query", string(query)), zap.String("index", types.ResourceFindingsIndex))
		return nil, err
	}

	result := make(map[string]types.SeverityResultWithTotal)
	for _, benchmarkBucket := range response.Aggregations.ComplianceResults.ConformanceFilter.BenchmarkGroup.Buckets {
		severityResult := types.SeverityResultWithTotal{}
		for _, severityBucket := range benchmarkBucket.SeverityGroup.Buckets {
			severityResult.TotalCount += severityBucket.ResourceCount.DocCount

			switch types.ParseComplianceResultSeverity(strings.ToLower(severityBucket.Key)) {
			case types.ComplianceResultSeverityCritical:
				severityResult.CriticalCount += severityBucket.ResourceCount.DocCount
			case types.ComplianceResultSeverityHigh:
				severityResult.HighCount += severityBucket.ResourceCount.DocCount
			case types.ComplianceResultSeverityMedium:
				severityResult.MediumCount += severityBucket.ResourceCount.DocCount
			case types.ComplianceResultSeverityLow:
				severityResult.LowCount += severityBucket.ResourceCount.DocCount
			case types.ComplianceResultSeverityNone, "":
				severityResult.NoneCount += severityBucket.ResourceCount.DocCount
			}
		}
		result[benchmarkBucket.Key] = severityResult
	}

	return result, nil
}

func GetPerBenchmarkResourceSeverityResultByJobId(ctx context.Context, logger *zap.Logger, client opengovernance.Client,
	benchmarkIDs []string, connectionIDs []string, resourceCollections []string,
	severities []types.ComplianceResultSeverity, conformanceStatuses []types.ConformanceStatus, summaryJobIDs string) (map[string]types.SeverityResultWithTotal, error) {
	request := make(map[string]any)
	filters := make([]map[string]any, 0)
	nestedFilters := make([]map[string]any, 0)
	if len(benchmarkIDs) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string][]string{
				"complianceResults.benchmarkID": benchmarkIDs,
			},
		})
	}
	if len(connectionIDs) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string][]string{
				"complianceResults.connectionID": connectionIDs,
			},
		})
	}
	if len(resourceCollections) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{
				"resourceCollections": resourceCollections,
			},
		})
	}
	filters = append(filters, map[string]any{
		"terms": map[string][]string{
			"jobId": {summaryJobIDs},
		},
	})

	if len(severities) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string]any{
				"complianceResults.severity": severities,
			},
		})
	}
	if len(conformanceStatuses) == 0 {
		conformanceStatuses = types.GetConformanceStatuses()
	}
	nestedFilters = append(nestedFilters, map[string]any{
		"terms": map[string]any{
			"complianceResults.conformanceStatus": conformanceStatuses,
		},
	})

	requestQuery := make(map[string]any, 0)
	if len(nestedFilters) > 0 {
		filters = append(filters, map[string]any{
			"nested": map[string]any{
				"path":  "complianceResults",
				"query": map[string]any{"bool": map[string]any{"filter": nestedFilters}},
			},
		})
	}
	if len(filters) > 0 {
		requestQuery["bool"] = map[string]any{
			"filter": filters,
		}
	}
	if len(requestQuery) > 0 {
		request["query"] = requestQuery
	}
	request["size"] = 0

	request["aggs"] = map[string]any{
		"complianceResults": map[string]any{
			"nested": map[string]any{
				"path": "complianceResults",
			},
			"aggs": map[string]any{
				"conformanceFilter": map[string]any{
					"filter": map[string]any{
						"terms": map[string]any{
							"complianceResults.conformanceStatus": conformanceStatuses,
						},
					},
					"aggs": map[string]any{
						"benchmarkGroup": map[string]any{
							"terms": map[string]any{
								"field": "complianceResults.benchmarkID",
								"size":  10000,
							},
							"aggs": map[string]any{
								"severityGroup": map[string]any{
									"terms": map[string]any{
										"field": "complianceResults.severity",
										"size":  10000,
									},
									"aggs": map[string]any{
										"resourceCount": map[string]any{
											"reverse_nested": map[string]any{},
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
		logger.Error("GetPerBenchmarkResourceSeverityResult", zap.Error(err), zap.Any("request", request))
	}

	logger.Info("GetPerBenchmarkResourceSeverityResult", zap.String("query", string(query)), zap.String("index", types.ResourceFindingsIndex))
	var response GetPerBenchmarkResourceSeverityResultResponse
	err = client.Search(ctx, types.ResourceFindingsIndex, string(query), &response)
	if err != nil {
		logger.Error("GetPerBenchmarkResourceSeverityResult", zap.Error(err), zap.String("query", string(query)), zap.String("index", types.ResourceFindingsIndex))
		return nil, err
	}

	result := make(map[string]types.SeverityResultWithTotal)
	for _, benchmarkBucket := range response.Aggregations.ComplianceResults.ConformanceFilter.BenchmarkGroup.Buckets {
		severityResult := types.SeverityResultWithTotal{}
		for _, severityBucket := range benchmarkBucket.SeverityGroup.Buckets {
			severityResult.TotalCount += severityBucket.ResourceCount.DocCount

			switch types.ParseComplianceResultSeverity(strings.ToLower(severityBucket.Key)) {
			case types.ComplianceResultSeverityCritical:
				severityResult.CriticalCount += severityBucket.ResourceCount.DocCount
			case types.ComplianceResultSeverityHigh:
				severityResult.HighCount += severityBucket.ResourceCount.DocCount
			case types.ComplianceResultSeverityMedium:
				severityResult.MediumCount += severityBucket.ResourceCount.DocCount
			case types.ComplianceResultSeverityLow:
				severityResult.LowCount += severityBucket.ResourceCount.DocCount
			case types.ComplianceResultSeverityNone, "":
				severityResult.NoneCount += severityBucket.ResourceCount.DocCount
			}
		}
		result[benchmarkBucket.Key] = severityResult
	}

	return result, nil
}

type GetPerFieldResourceConformanceResultResponse struct {
	Aggregations struct {
		ComplianceResults struct {
			ConformanceFilter struct {
				FieldGroup struct {
					Buckets []struct {
						Key              string `json:"key"`
						ConformanceGroup struct {
							Buckets []struct {
								Key           string `json:"key"`
								ResourceCount struct {
									DocCount int `json:"doc_count"`
								} `json:"resourceCount"`
							} `json:"buckets"`
						} `json:"conformanceGroup"`
					} `json:"buckets"`
				} `json:"fieldGroup"`
			} `json:"conformanceFilter"`
		} `json:"complianceResults"`
	} `json:"aggregations"`
}

// GetPerFieldResourceConformanceResult
// field could be: connectionID, benchmarkID, controlID, severity, conformanceStatus
func GetPerFieldResourceConformanceResult(ctx context.Context, logger *zap.Logger, client opengovernance.Client,
	field string,
	connectionIDs []string, notConnectionIDs []string,
	resourceCollections []string,
	controlIDs []string, benchmarkIDs []string,
	severities []types.ComplianceResultSeverity, conformanceStatuses []types.ConformanceStatus, startTime, endTime *time.Time) (map[string]types.ConformanceStatusSummaryWithTotal, error) {
	if field != "connectionID" && field != "benchmarkID" && field != "controlID" && field != "severity" && field != "conformanceStatus" {
		return nil, fmt.Errorf("field %s is not supported", field)
	}
	request := make(map[string]any)
	filters := make([]map[string]any, 0)
	nestedFilters := make([]map[string]any, 0)
	if len(connectionIDs) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string][]string{
				"complianceResults.connectionID": connectionIDs,
			},
		})
	}
	if len(notConnectionIDs) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"bool": map[string]any{
				"must_not": []map[string]any{
					{
						"terms": map[string][]string{
							"complianceResults.connectionID": notConnectionIDs,
						},
					},
				},
			},
		})
	}
	if len(controlIDs) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string][]string{
				"complianceResults.controlID": controlIDs,
			},
		})
	}
	if len(benchmarkIDs) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string][]string{
				"complianceResults.benchmarkID": benchmarkIDs,
			},
		})
	}
	if len(resourceCollections) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{
				"resourceCollections": resourceCollections,
			},
		})
	}
	if endTime != nil && startTime != nil {
		filters = append(filters, map[string]any{
			"range": map[string]any{
				"evaluatedAt": map[string]any{
					"gte": startTime.UnixMilli(),
					"lte": endTime.UnixMilli(),
				},
			},
		})
	} else if endTime != nil {
		filters = append(filters, map[string]any{
			"range": map[string]any{
				"evaluatedAt": map[string]any{
					"lte": endTime.UnixMilli(),
				},
			},
		})
	} else if startTime != nil {
		filters = append(filters, map[string]any{
			"range": map[string]any{
				"evaluatedAt": map[string]any{
					"gte": startTime.UnixMilli(),
				},
			},
		})
	}
	if len(severities) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string]any{
				"complianceResults.severity": severities,
			},
		})
	}
	if len(conformanceStatuses) == 0 {
		conformanceStatuses = types.GetConformanceStatuses()
	}
	nestedFilters = append(nestedFilters, map[string]any{
		"terms": map[string]any{
			"complianceResults.conformanceStatus": conformanceStatuses,
		},
	})

	requestQuery := make(map[string]any)

	nestedQuery := map[string]any{
		"path":  "complianceResults",
		"query": map[string]any{"bool": map[string]any{"filter": nestedFilters}},
	}

	if len(filters) > 0 {
		requestQuery["bool"] = map[string]any{
			"must": []map[string]any{
				{"nested": nestedQuery},
				{"bool": map[string]any{"filter": filters}},
			},
		}
	} else if len(nestedFilters) > 0 {
		requestQuery["nested"] = nestedQuery
	}
	if len(requestQuery) > 0 {
		request["query"] = requestQuery
	}
	request["size"] = 0

	request["aggs"] = map[string]any{
		"complianceResults": map[string]any{
			"nested": map[string]any{
				"path": "complianceResults",
			},
			"aggs": map[string]any{
				"conformanceFilter": map[string]any{
					"filter": map[string]any{
						"terms": map[string]any{
							"complianceResults.conformanceStatus": conformanceStatuses,
						},
					},
					"aggs": map[string]any{
						"fieldGroup": map[string]any{
							"terms": map[string]any{
								"field": fmt.Sprintf("complianceResults.%s", field),
								"size":  10000,
							},
							"aggs": map[string]any{
								"conformanceGroup": map[string]any{
									"terms": map[string]any{
										"field": "complianceResults.conformanceStatus",
										"size":  10000,
									},
									"aggs": map[string]any{
										"resourceCount": map[string]any{
											"reverse_nested": map[string]any{},
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
		logger.Error("GetPerFieldResourceConformanceResult", zap.Error(err), zap.Any("request", request))
		return nil, err
	}

	logger.Info("GetPerFieldResourceConformanceResult", zap.String("query", string(query)), zap.String("index", types.ResourceFindingsIndex))
	var response GetPerFieldResourceConformanceResultResponse
	err = client.Search(ctx, types.ResourceFindingsIndex, string(query), &response)
	if err != nil {
		logger.Error("GetPerFieldResourceConformanceResult", zap.Error(err), zap.String("query", string(query)), zap.String("index", types.ResourceFindingsIndex))
		return nil, err
	}

	result := make(map[string]types.ConformanceStatusSummaryWithTotal)
	for _, connectionBucket := range response.Aggregations.ComplianceResults.ConformanceFilter.FieldGroup.Buckets {
		conformanceStatusSummary := types.ConformanceStatusSummaryWithTotal{}
		for _, conformanceBucket := range connectionBucket.ConformanceGroup.Buckets {
			conformanceStatusSummary.TotalCount += conformanceBucket.ResourceCount.DocCount

			switch types.ParseConformanceStatus(strings.ToLower(conformanceBucket.Key)) {
			case types.ConformanceStatusOK:
				conformanceStatusSummary.OkCount += conformanceBucket.ResourceCount.DocCount
			case types.ConformanceStatusALARM:
				conformanceStatusSummary.AlarmCount += conformanceBucket.ResourceCount.DocCount
			case types.ConformanceStatusINFO:
				conformanceStatusSummary.InfoCount += conformanceBucket.ResourceCount.DocCount
			case types.ConformanceStatusSKIP:
				conformanceStatusSummary.SkipCount += conformanceBucket.ResourceCount.DocCount
			case types.ConformanceStatusERROR:
				conformanceStatusSummary.ErrorCount += conformanceBucket.ResourceCount.DocCount
			}
		}
		result[connectionBucket.Key] = conformanceStatusSummary
	}

	return result, nil
}

func GetPerFieldTopWithIssues(ctx context.Context, logger *zap.Logger, client opengovernance.Client,
	field string,
	connectionIDs []string, notConnectionIDs []string,
	resourceCollections []string,
	controlIDs []string, benchmarkIDs []string,
	severities []types.ComplianceResultSeverity, topCount int) (map[string]types.ConformanceStatusSummaryWithTotal, error) {
	if field != "connectionID" && field != "benchmarkID" && field != "controlID" && field != "severity" && field != "conformanceStatus" &&
		field != "resourceType" && field != "resourceID" {
		return nil, fmt.Errorf("field %s is not supported", field)
	}
	request := make(map[string]any)
	filters := make([]map[string]any, 0)
	nestedFilters := make([]map[string]any, 0)
	if len(connectionIDs) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string][]string{
				"complianceResults.connectionID": connectionIDs,
			},
		})
	}
	if len(notConnectionIDs) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"bool": map[string]any{
				"must_not": []map[string]any{
					{
						"terms": map[string][]string{
							"complianceResults.connectionID": notConnectionIDs,
						},
					},
				},
			},
		})
	}
	if len(controlIDs) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string][]string{
				"complianceResults.controlID": controlIDs,
			},
		})
	}
	if len(benchmarkIDs) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string][]string{
				"complianceResults.benchmarkID": benchmarkIDs,
			},
		})
	}
	if len(resourceCollections) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{
				"resourceCollections": resourceCollections,
			},
		})
	}
	if len(severities) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string]any{
				"complianceResults.severity": severities,
			},
		})
	}
	conformanceStatuses := []types.ConformanceStatus{types.ConformanceStatusALARM}

	nestedFilters = append(nestedFilters, map[string]any{
		"terms": map[string]any{
			"complianceResults.conformanceStatus": conformanceStatuses,
		},
	})

	requestQuery := make(map[string]any, 0)
	if len(nestedFilters) > 0 {
		requestQuery["nested"] = map[string]any{
			"path":  "complianceResults",
			"query": map[string]any{"bool": map[string]any{"filter": nestedFilters}},
		}
	}
	if len(filters) > 0 {
		requestQuery["bool"] = map[string]any{
			"filter": filters,
		}
	}
	if len(requestQuery) > 0 {
		request["query"] = requestQuery
	}
	request["size"] = 0

	request["aggs"] = map[string]any{
		"complianceResults": map[string]any{
			"nested": map[string]any{
				"path": "complianceResults",
			},
			"aggs": map[string]any{
				"conformanceFilter": map[string]any{
					"filter": map[string]any{
						"terms": map[string]any{
							"complianceResults.conformanceStatus": conformanceStatuses,
						},
					},
					"aggs": map[string]any{
						"fieldGroup": map[string]any{
							"terms": map[string]any{
								"field": fmt.Sprintf("complianceResults.%s", field),
								"size":  topCount,
								"order": map[string]any{
									"conformanceGroup>resourceCount.doc_count": "desc",
								},
							},
							"aggs": map[string]any{
								"conformanceGroup": map[string]any{
									"filter": map[string]any{
										"terms": map[string]any{
											"complianceResults.conformanceStatus": conformanceStatuses,
										},
									},
									"aggs": map[string]any{
										"resourceCount": map[string]any{
											"reverse_nested": map[string]any{},
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
		logger.Error("Error while marshaling es request", zap.Error(err))
		return nil, err
	}

	logger.Info("request for GetPerFieldTopWithIssues", zap.String("request", string(query)))
	var response GetPerFieldTopWithIssuesResponse
	err = client.Search(ctx, types.ResourceFindingsIndex, string(query), &response)
	if err != nil {
		logger.Error("Error while searching es request", zap.Error(err))
		return nil, err
	}

	result := make(map[string]types.ConformanceStatusSummaryWithTotal)
	for _, connectionBucket := range response.Aggregations.ComplianceResults.ConformanceFilter.FieldGroup.Buckets {
		conformanceStatusSummary := types.ConformanceStatusSummaryWithTotal{
			TotalCount: connectionBucket.ConformanceGroup.ResourceCount.DocCount,
			ConformanceStatusSummary: types.ConformanceStatusSummary{
				AlarmCount: connectionBucket.ConformanceGroup.ResourceCount.DocCount,
			},
		}

		result[connectionBucket.Key] = conformanceStatusSummary
	}

	return result, nil
}

type GetPerFieldTopWithIssuesResponse struct {
	Aggregations struct {
		ComplianceResults struct {
			ConformanceFilter struct {
				FieldGroup struct {
					Buckets []struct {
						Key              string `json:"key"`
						ConformanceGroup struct {
							ResourceCount struct {
								DocCount int `json:"doc_count"`
							} `json:"resourceCount"`
						} `json:"conformanceGroup"`
					} `json:"buckets"`
				} `json:"fieldGroup"`
			} `json:"conformanceFilter"`
		} `json:"complianceResults"`
	} `json:"aggregations"`
}
