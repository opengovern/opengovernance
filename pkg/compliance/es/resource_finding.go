package es

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"go.uber.org/zap"
	"strings"
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
		Total kaytu.SearchTotal          `json:"total"`
		Hits  []ResourceFindingsQueryHit `json:"hits"`
	} `json:"hits"`
}

func DeleteOtherResourceFindingsExcept(logger *zap.Logger, client kaytu.Client, kaytuResourceIDs []string, lessThanJobId uint) error {
	queryMap := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must_not": map[string]any{
					"terms": map[string]any{
						"kaytuResourceID": kaytuResourceIDs,
					},
				},
				"filter": map[string]any{
					"range": map[string]any{
						"jobId": map[string]any{
							"lt": lessThanJobId,
						},
					},
				},
			},
		},
	}
	if len(kaytuResourceIDs) == 0 {
		delete(queryMap["query"].(map[string]any)["bool"].(map[string]any), "must_not")
	}

	query, err := json.Marshal(queryMap)
	if err != nil {
		logger.Error("failed to marshal query", zap.Error(err))
		return err
	}

	es := client.ES()
	_, err = es.DeleteByQuery(
		[]string{types.ResourceFindingsIndex},
		bytes.NewReader(query),
		es.DeleteByQuery.WithContext(context.TODO()),
	)
	if err != nil {
		logger.Error("failed to delete old resource findings", zap.Error(err), zap.String("query", string(query)), zap.Uint("lessThanJobId", lessThanJobId))
		return err
	}

	return nil
}

func ResourceFindingsQuery(logger *zap.Logger, client kaytu.Client,
	connector []source.Type, connectionID []string, resourceCollection []string,
	resourceTypes []string,
	benchmarkID []string, controlID []string,
	severity []types.FindingSeverity, conformanceStatuses []types.ConformanceStatus,
	sorts []api.ResourceFindingsSort, pageSizeLimit int, searchAfter []any) ([]ResourceFindingsQueryHit, int64, error) {

	nestedFilters := make([]map[string]any, 0)
	if len(connector) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string]any{
				"findings.connector": connector,
			},
		})
	}
	if len(connectionID) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string]any{
				"findings.connectionID": connectionID,
			},
		})
	}
	if len(benchmarkID) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string]any{
				"findings.benchmarkID": benchmarkID,
			},
		})
	}
	if len(controlID) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string]any{
				"findings.controlID": controlID,
			},
		})
	}
	if len(severity) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string]any{
				"findings.severity": severity,
			},
		})
	}
	if len(conformanceStatuses) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string]any{
				"findings.conformanceStatus": conformanceStatuses,
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

	if len(nestedFilters) > 0 {
		filters = append(filters, map[string]any{"nested": map[string]any{
			"path": "findings",
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
		case sort.KaytuResourceID != nil:
			requestSort = append(requestSort, map[string]any{
				"kaytuResourceID": *sort.KaytuResourceID,
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
		case sort.FailedCount != nil:
			scriptSource :=
				fmt.Sprintf(`int total = 0; 
for (int i=0; i<params['_source']['findings'].length;++i) { 
  if(params['_source']['findings'][i]['conformanceStatus'] != '%s') 
    total+=1;
  } 
return total;`, types.ConformanceStatusOK)
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
	err = client.SearchWithTrackTotalHits(context.Background(), types.ResourceFindingsIndex, string(request), nil, &response, true)
	if err != nil {
		return nil, 0, err
	}

	return response.Hits.Hits, response.Hits.Total.Value, nil
}

type GetPerBenchmarkResourceSeverityResultResponse struct {
	Aggregations struct {
		Findings struct {
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
		} `json:"findings"`
	} `json:"aggregations"`
}

func GetPerBenchmarkResourceSeverityResult(logger *zap.Logger, client kaytu.Client,
	benchmarkIDs []string, connectionIDs []string, resourceCollections []string,
	severities []types.FindingSeverity, conformanceStatuses []types.ConformanceStatus) (map[string]types.SeverityResultWithTotal, error) {
	request := make(map[string]any)
	filters := make([]map[string]any, 0)
	nestedFilters := make([]map[string]any, 0)
	if len(benchmarkIDs) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string][]string{
				"findings.benchmarkID": benchmarkIDs,
			},
		})
	}
	if len(connectionIDs) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string][]string{
				"findings.connectionID": connectionIDs,
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
				"findings.severity": severities,
			},
		})
	}
	if len(conformanceStatuses) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string]any{
				"findings.conformanceStatus": conformanceStatuses,
			},
		})
	}

	requestQuery := make(map[string]any, 0)
	if len(nestedFilters) > 0 {
		filters = append(filters, map[string]any{
			"nested": map[string]any{
				"path":  "findings",
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
		"findings": map[string]any{
			"nested": map[string]any{
				"path": "findings",
			},
			"aggs": map[string]any{
				"benchmarkGroup": map[string]any{
					"terms": map[string]any{
						"field": "findings.benchmarkID",
						"size":  10000,
					},
					"aggs": map[string]any{
						"severityGroup": map[string]any{
							"terms": map[string]any{
								"field": "findings.severity",
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
	}

	query, err := json.Marshal(request)
	if err != nil {
		logger.Error("GetPerBenchmarkResourceSeverityResult", zap.Error(err), zap.Any("request", request))
	}

	logger.Info("GetPerBenchmarkResourceSeverityResult", zap.String("query", string(query)), zap.String("index", types.ResourceFindingsIndex))
	var response GetPerBenchmarkResourceSeverityResultResponse
	err = client.Search(context.Background(), types.ResourceFindingsIndex, string(query), &response)
	if err != nil {
		logger.Error("GetPerBenchmarkResourceSeverityResult", zap.Error(err), zap.String("query", string(query)), zap.String("index", types.ResourceFindingsIndex))
		return nil, err
	}

	result := make(map[string]types.SeverityResultWithTotal)
	for _, benchmarkBucket := range response.Aggregations.Findings.BenchmarkGroup.Buckets {
		severityResult := types.SeverityResultWithTotal{}
		for _, severityBucket := range benchmarkBucket.SeverityGroup.Buckets {
			severityResult.TotalCount += severityBucket.ResourceCount.DocCount

			switch types.ParseFindingSeverity(strings.ToLower(severityBucket.Key)) {
			case types.FindingSeverityCritical:
				severityResult.CriticalCount += severityBucket.ResourceCount.DocCount
			case types.FindingSeverityHigh:
				severityResult.HighCount += severityBucket.ResourceCount.DocCount
			case types.FindingSeverityMedium:
				severityResult.MediumCount += severityBucket.ResourceCount.DocCount
			case types.FindingSeverityLow:
				severityResult.LowCount += severityBucket.ResourceCount.DocCount
			case types.FindingSeverityNone, "":
				severityResult.NoneCount += severityBucket.ResourceCount.DocCount
			}
		}
		result[benchmarkBucket.Key] = severityResult
	}

	return result, nil
}

type GetPerFieldResourceConformanceResultResponse struct {
	Aggregations struct {
		Findings struct {
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
		} `json:"findings"`
	} `json:"aggregations"`
}

// GetPerFieldResourceConformanceResult
// field could be: connectionID, benchmarkID, controlID, severity, conformanceStatus
func GetPerFieldResourceConformanceResult(logger *zap.Logger, client kaytu.Client,
	field string,
	connectionIDs []string, resourceCollections []string,
	controlIDs []string, benchmarkIDs []string,
	severities []types.FindingSeverity, conformanceStatuses []types.ConformanceStatus) (map[string]types.ConformanceStatusSummaryWithTotal, error) {
	if field != "connectionID" && field != "benchmarkID" && field != "controlID" && field != "severity" && field != "conformanceStatus" {
		return nil, fmt.Errorf("field %s is not supported", field)
	}
	request := make(map[string]any)
	filters := make([]map[string]any, 0)
	nestedFilters := make([]map[string]any, 0)
	if len(connectionIDs) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string][]string{
				"findings.connectionID": connectionIDs,
			},
		})
	}
	if len(controlIDs) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string][]string{
				"findings.controlID": controlIDs,
			},
		})
	}
	if len(benchmarkIDs) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string][]string{
				"findings.benchmarkID": benchmarkIDs,
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
				"findings.severity": severities,
			},
		})
	}
	if len(conformanceStatuses) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string]any{
				"findings.conformanceStatus": conformanceStatuses,
			},
		})
	}

	requestQuery := make(map[string]any, 0)
	if len(nestedFilters) > 0 {
		requestQuery["nested"] = map[string]any{
			"path":  "findings",
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
		"findings": map[string]any{
			"nested": map[string]any{
				"path": "findings",
			},
			"aggs": map[string]any{
				"fieldGroup": map[string]any{
					"terms": map[string]any{
						"field": fmt.Sprintf("findings.%s", field),
						"size":  10000,
					},
					"aggs": map[string]any{
						"conformanceGroup": map[string]any{
							"terms": map[string]any{
								"field": "findings.conformanceStatus",
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
	}

	query, err := json.Marshal(request)
	if err != nil {
		logger.Error("GetPerFieldResourceConformanceResult", zap.Error(err), zap.Any("request", request))
		return nil, err
	}

	logger.Info("GetPerFieldResourceConformanceResult", zap.String("query", string(query)), zap.String("index", types.ResourceFindingsIndex))
	var response GetPerFieldResourceConformanceResultResponse
	err = client.Search(context.Background(), types.ResourceFindingsIndex, string(query), &response)
	if err != nil {
		logger.Error("GetPerFieldResourceConformanceResult", zap.Error(err), zap.String("query", string(query)), zap.String("index", types.ResourceFindingsIndex))
		return nil, err
	}

	result := make(map[string]types.ConformanceStatusSummaryWithTotal)
	for _, connectionBucket := range response.Aggregations.Findings.FieldGroup.Buckets {
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
