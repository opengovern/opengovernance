package es

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/open-governance/pkg/compliance/api"
	"github.com/kaytu-io/open-governance/pkg/types"
	"go.uber.org/zap"
	"time"

	"github.com/kaytu-io/kaytu-util/pkg/source"

	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
)

type FindingsQueryResponse struct {
	Hits struct {
		Total kaytu.SearchTotal  `json:"total"`
		Hits  []FindingsQueryHit `json:"hits"`
	} `json:"hits"`
	PitID string `json:"pit_id"`
}

type FindingsQueryHit struct {
	ID      string        `json:"_id"`
	Score   float64       `json:"_score"`
	Index   string        `json:"_index"`
	Type    string        `json:"_type"`
	Version int64         `json:"_version,omitempty"`
	Source  types.Finding `json:"_source"`
	Sort    []any         `json:"sort"`
}

type FindingPaginator struct {
	paginator *kaytu.BaseESPaginator
}

func NewFindingPaginator(client kaytu.Client, idx string, filters []kaytu.BoolFilter, limit *int64, sort []map[string]any) (FindingPaginator, error) {
	paginator, err := kaytu.NewPaginatorWithSort(client.ES(), idx, filters, limit, sort)
	if err != nil {
		return FindingPaginator{}, err
	}

	p := FindingPaginator{
		paginator: paginator,
	}

	return p, nil
}

func (p FindingPaginator) HasNext() bool {
	return !p.paginator.Done()
}

func (p FindingPaginator) Close(ctx context.Context) error {
	return p.paginator.Deallocate(ctx)
}

func (p FindingPaginator) NextPage(ctx context.Context) ([]types.Finding, error) {
	var response FindingsQueryResponse
	err := p.paginator.SearchWithLog(ctx, &response, true)
	if err != nil {
		return nil, err
	}

	var values []types.Finding
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

type FindingsCountQueryHit struct {
	Aggregations struct {
		ControlIDCount struct {
			Buckets []struct {
				Key                    string `json:"key"`
				DocCount               int64  `json:"doc_count"`
				ConformanceStatusCount struct {
					Buckets []struct {
						Key      string `json:"key"`
						DocCount int64  `json:"doc_count"`
					} `json:"buckets"`
				} `json:"conformanceStatus_count"`
			} `json:"buckets"`
		} `json:"controlID_count"`
	} `json:"aggregations"`
}

func FindingsCountByControlID(ctx context.Context, logger *zap.Logger, client kaytu.Client, resourceIDs []string, provider []source.Type, connectionID []string, notConnectionID []string, resourceTypes []string, benchmarkID []string, controlID []string, severity []types.FindingSeverity, lastTransitionFrom *time.Time, lastTransitionTo *time.Time, evaluatedAtFrom *time.Time, evaluatedAtTo *time.Time, stateActive []bool, conformanceStatuses []types.ConformanceStatus) (map[string]map[string]int64, error) {
	idx := types.FindingsIndex
	var filters []kaytu.BoolFilter
	if len(resourceIDs) > 0 {
		filters = append(filters, kaytu.NewTermsFilter("resourceID", resourceIDs))
	}
	if len(resourceTypes) > 0 {
		filters = append(filters, kaytu.NewTermsFilter("resourceType", resourceTypes))
	}
	if len(benchmarkID) > 0 {
		filters = append(filters, kaytu.NewTermsFilter("parentBenchmarks", benchmarkID))
	}
	if len(controlID) > 0 {
		filters = append(filters, kaytu.NewTermsFilter("controlID", controlID))
	}
	if len(severity) > 0 {
		strSeverity := make([]string, 0)
		for _, s := range severity {
			strSeverity = append(strSeverity, string(s))
		}
		filters = append(filters, kaytu.NewTermsFilter("severity", strSeverity))
	}
	if len(conformanceStatuses) > 0 {
		strConformanceStatus := make([]string, 0)
		for _, cr := range conformanceStatuses {
			strConformanceStatus = append(strConformanceStatus, string(cr))
		}
		filters = append(filters, kaytu.NewTermsFilter("conformanceStatus", strConformanceStatus))
	}
	if len(connectionID) > 0 {
		filters = append(filters, kaytu.NewTermsFilter("connectionID", connectionID))
	}
	if len(notConnectionID) > 0 {
		filters = append(filters, kaytu.NewBoolMustNotFilter(kaytu.NewTermsFilter("connectionID", notConnectionID)))
	}
	if len(provider) > 0 {
		var connectors []string
		for _, p := range provider {
			connectors = append(connectors, p.String())
		}
		filters = append(filters, kaytu.NewTermsFilter("connector", connectors))
	}
	if len(stateActive) > 0 {
		strStateActive := make([]string, 0)
		for _, s := range stateActive {
			strStateActive = append(strStateActive, fmt.Sprintf("%v", s))
		}
		filters = append(filters, kaytu.NewTermsFilter("stateActive", strStateActive))
	}
	if lastTransitionFrom != nil && lastTransitionTo != nil {
		filters = append(filters, kaytu.NewRangeFilter("lastTransition",
			"", fmt.Sprintf("%d", lastTransitionFrom.UnixMilli()),
			"", fmt.Sprintf("%d", lastTransitionTo.UnixMilli())))
	} else if lastTransitionFrom != nil {
		filters = append(filters, kaytu.NewRangeFilter("lastTransition",
			"", fmt.Sprintf("%d", lastTransitionFrom.UnixMilli()),
			"", ""))
	} else if lastTransitionTo != nil {
		filters = append(filters, kaytu.NewRangeFilter("lastTransition",
			"", "",
			"", fmt.Sprintf("%d", lastTransitionTo.UnixMilli())))
	}
	if evaluatedAtFrom != nil && evaluatedAtTo != nil {
		filters = append(filters, kaytu.NewRangeFilter("evaluatedAt",
			"", fmt.Sprintf("%d", evaluatedAtFrom.UnixMilli()),
			"", fmt.Sprintf("%d", evaluatedAtTo.UnixMilli())))
	} else if evaluatedAtFrom != nil {
		filters = append(filters, kaytu.NewRangeFilter("evaluatedAt",
			"", fmt.Sprintf("%d", evaluatedAtFrom.UnixMilli()),
			"", ""))
	} else if evaluatedAtTo != nil {
		filters = append(filters, kaytu.NewRangeFilter("evaluatedAt",
			"", "",
			"", fmt.Sprintf("%d", evaluatedAtTo.UnixMilli())))
	}

	query := map[string]any{
		"size": 0,
		"query": map[string]any{
			"bool": map[string]any{
				"filter": filters,
			},
		},
		"aggs": map[string]any{
			"controlID_count": map[string]any{
				"terms": map[string]any{
					"field": "controlID",
					"size":  10000,
				},
				"aggs": map[string]any{
					"conformanceStatus_count": map[string]any{
						"terms": map[string]any{
							"field": "conformanceStatus",
							"size":  10,
						},
					},
				},
			},
		},
	}

	queryJson, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}

	logger.Info("FindingsCountByControlID", zap.String("query", string(queryJson)), zap.String("index", idx))

	var response FindingsCountQueryHit
	err = client.SearchWithTrackTotalHits(ctx, idx, string(queryJson), nil, &response, true)
	if err != nil {
		return nil, err
	}

	controlIDCount := make(map[string]map[string]int64)
	for _, bucket := range response.Aggregations.ControlIDCount.Buckets {
		controlIDCount[bucket.Key] = make(map[string]int64)
		for _, conformanceBucket := range bucket.ConformanceStatusCount.Buckets {
			controlIDCount[bucket.Key][conformanceBucket.Key] = conformanceBucket.DocCount
		}
	}

	return controlIDCount, nil
}

func FindingsQuery(ctx context.Context, logger *zap.Logger, client kaytu.Client, resourceIDs []string, provider []source.Type,
	connectionID []string, notConnectionID []string, resourceTypes []string, benchmarkID []string, controlID []string,
	severity []types.FindingSeverity, lastTransitionFrom *time.Time, lastTransitionTo *time.Time,
	evaluatedAtFrom *time.Time, evaluatedAtTo *time.Time, stateActive []bool, conformanceStatuses []types.ConformanceStatus,
	sorts []api.FindingsSort, pageSizeLimit int, searchAfter []any, jobIDs []string) ([]FindingsQueryHit, int64, error) {
	idx := types.FindingsIndex

	requestSort := make([]map[string]any, 0, len(sorts)+1)
	for _, sort := range sorts {
		switch {
		case sort.Connector != nil:
			requestSort = append(requestSort, map[string]any{
				"connector": *sort.Connector,
			})
		case sort.KaytuResourceID != nil:
			requestSort = append(requestSort, map[string]any{
				"kaytuResourceID": *sort.KaytuResourceID,
			})
		case sort.ResourceID != nil:
			requestSort = append(requestSort, map[string]any{
				"resourceID": *sort.ResourceID,
			})
		case sort.ResourceTypeID != nil:
			requestSort = append(requestSort, map[string]any{
				"resourceType": *sort.ResourceTypeID,
			})
		case sort.ConnectionID != nil:
			requestSort = append(requestSort, map[string]any{
				"connectionID": *sort.ConnectionID,
			})
		case sort.BenchmarkID != nil:
			requestSort = append(requestSort, map[string]any{
				"benchmarkID": *sort.BenchmarkID,
			})
		case sort.ControlID != nil:
			requestSort = append(requestSort, map[string]any{
				"controlID": *sort.ControlID,
			})
		case sort.Severity != nil:
			scriptSource :=
				`if (params['_source']['severity'] == 'critical') {
					return 5
				} else if (params['_source']['severity'] == 'high') {
					return 4
				} else if (params['_source']['severity'] == 'medium') {
					return 3
				} else if (params['_source']['severity'] == 'low') {
					return 2
				} else if (params['_source']['severity'] == 'none') {
					return 1
				} else {
					return 1
				}`
			requestSort = append(requestSort, map[string]any{
				"_script": map[string]any{
					"type": "number",
					"script": map[string]any{
						"lang":   "painless",
						"source": scriptSource,
					},
					"order": *sort.Severity,
				},
			})
		case sort.ConformanceStatus != nil:
			scriptSource :=
				`if (params['_source']['conformanceStatus'] == 'alarm') {
					return 5
				} else if (params['_source']['conformanceStatus'] == 'error') {
					return 4
				} else if (params['_source']['conformanceStatus'] == 'info') {
					return 3
				} else if (params['_source']['conformanceStatus'] == 'skip') {
					return 2
				} else if (params['_source']['conformanceStatus'] == 'ok') {
					return 1
				} else {
					return 1
				}`
			requestSort = append(requestSort, map[string]any{
				"_script": map[string]any{
					"type": "number",
					"script": map[string]any{
						"lang":   "painless",
						"source": scriptSource,
					},
					"order": *sort.ConformanceStatus,
				},
			})
		case sort.StateActive != nil:
			requestSort = append(requestSort, map[string]any{
				"stateActive": *sort.StateActive,
			})
		}
	}
	requestSort = append(requestSort, map[string]any{
		"_id": "asc",
	})

	var filters []kaytu.BoolFilter
	if len(resourceIDs) > 0 {
		filters = append(filters, kaytu.NewTermsFilter("resourceID", resourceIDs))
	}
	if len(resourceTypes) > 0 {
		filters = append(filters, kaytu.NewTermsFilter("resourceType", resourceTypes))
	}
	if len(benchmarkID) > 0 {
		filters = append(filters, kaytu.NewTermsFilter("parentBenchmarks", benchmarkID))
	}
	if len(controlID) > 0 {
		filters = append(filters, kaytu.NewTermsFilter("controlID", controlID))
	}
	if len(jobIDs) > 0 {
		filters = append(filters, kaytu.NewTermsFilter("parentComplianceJobID", jobIDs))
	}
	if len(severity) > 0 {
		strSeverity := make([]string, 0)
		for _, s := range severity {
			strSeverity = append(strSeverity, string(s))
		}
		filters = append(filters, kaytu.NewTermsFilter("severity", strSeverity))
	}
	if len(conformanceStatuses) > 0 {
		strConformanceStatus := make([]string, 0)
		for _, cr := range conformanceStatuses {
			strConformanceStatus = append(strConformanceStatus, string(cr))
		}
		filters = append(filters, kaytu.NewTermsFilter("conformanceStatus", strConformanceStatus))
	}
	if len(connectionID) > 0 {
		filters = append(filters, kaytu.NewTermsFilter("connectionID", connectionID))
	}
	if len(notConnectionID) > 0 {
		filters = append(filters, kaytu.NewBoolMustNotFilter(kaytu.NewTermsFilter("connectionID", notConnectionID)))
	}
	if len(provider) > 0 {
		var connectors []string
		for _, p := range provider {
			connectors = append(connectors, p.String())
		}
		filters = append(filters, kaytu.NewTermsFilter("connector", connectors))
	}
	if len(stateActive) > 0 {
		strStateActive := make([]string, 0)
		for _, s := range stateActive {
			strStateActive = append(strStateActive, fmt.Sprintf("%v", s))
		}
		filters = append(filters, kaytu.NewTermsFilter("stateActive", strStateActive))
	}
	if lastTransitionFrom != nil && lastTransitionTo != nil {
		filters = append(filters, kaytu.NewRangeFilter("lastTransition",
			"", fmt.Sprintf("%d", lastTransitionFrom.UnixMilli()),
			"", fmt.Sprintf("%d", lastTransitionTo.UnixMilli())))
	} else if lastTransitionFrom != nil {
		filters = append(filters, kaytu.NewRangeFilter("lastTransition",
			"", fmt.Sprintf("%d", lastTransitionFrom.UnixMilli()),
			"", ""))
	} else if lastTransitionTo != nil {
		filters = append(filters, kaytu.NewRangeFilter("lastTransition",
			"", "",
			"", fmt.Sprintf("%d", lastTransitionTo.UnixMilli())))
	}
	if evaluatedAtFrom != nil && evaluatedAtTo != nil {
		filters = append(filters, kaytu.NewRangeFilter("evaluatedAt",
			"", fmt.Sprintf("%d", evaluatedAtFrom.UnixMilli()),
			"", fmt.Sprintf("%d", evaluatedAtTo.UnixMilli())))
	} else if evaluatedAtFrom != nil {
		filters = append(filters, kaytu.NewRangeFilter("evaluatedAt",
			"", fmt.Sprintf("%d", evaluatedAtFrom.UnixMilli()),
			"", ""))
	} else if evaluatedAtTo != nil {
		filters = append(filters, kaytu.NewRangeFilter("evaluatedAt",
			"", "",
			"", fmt.Sprintf("%d", evaluatedAtTo.UnixMilli())))
	}

	query := make(map[string]any)
	if len(filters) > 0 {
		query["query"] = map[string]any{
			"bool": map[string]any{
				"filter": filters,
			},
		}
	}
	query["sort"] = requestSort
	if len(searchAfter) > 0 {
		query["search_after"] = searchAfter
	}
	if pageSizeLimit == 0 {
		pageSizeLimit = 1000
	}
	query["size"] = pageSizeLimit
	queryJson, err := json.Marshal(query)
	if err != nil {
		return nil, 0, err
	}

	logger.Info("FindingsQuery", zap.String("query", string(queryJson)), zap.String("index", idx))

	var response FindingsQueryResponse
	err = client.SearchWithTrackTotalHits(ctx, idx, string(queryJson), nil, &response, true)
	if err != nil {
		return nil, 0, err
	}

	return response.Hits.Hits, response.Hits.Total.Value, err
}

type FindingsCountResponse struct {
	Hits  FindingsCountHits `json:"hits"`
	PitID string            `json:"pit_id"`
}
type FindingsCountHits struct {
	Total kaytu.SearchTotal `json:"total"`
}

func FindingsCount(ctx context.Context, client kaytu.Client, conformanceStatuses []types.ConformanceStatus, stateActive []bool) (int64, error) {
	idx := types.FindingsIndex

	filters := make([]map[string]any, 0)
	if len(conformanceStatuses) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string]any{
				"conformanceStatus": conformanceStatuses,
			},
		})
	}
	if len(stateActive) > 0 {
		strStateActive := make([]string, 0)
		for _, s := range stateActive {
			strStateActive = append(strStateActive, fmt.Sprintf("%v", s))
		}
		filters = append(filters, map[string]any{
			"terms": map[string]any{
				"stateActive": strStateActive,
			},
		})
	}

	query := make(map[string]any)
	query["size"] = 0

	if len(filters) > 0 {
		query["query"] = map[string]any{
			"bool": map[string]any{
				"filter": filters,
			},
		}
	}

	queryJson, err := json.Marshal(query)
	if err != nil {
		return 0, err
	}

	var response FindingsCountResponse
	err = client.SearchWithTrackTotalHits(ctx, idx, string(queryJson), nil, &response, true)
	if err != nil {
		return 0, err
	}

	return response.Hits.Total.Value, err
}

type AggregationResult struct {
	DocCountErrorUpperBound int `json:"doc_count_error_upper_bound"`
	SumOtherDocCount        int `json:"sum_other_doc_count"`
	Buckets                 []struct {
		Key      string `json:"key"`
		DocCount int    `json:"doc_count"`
	} `json:"buckets"`
}

func (a AggregationResult) GetBucketsKeys() []string {
	var keys []string
	for _, bucket := range a.Buckets {
		keys = append(keys, bucket.Key)
	}
	return keys
}

type FindingFiltersAggregationResponse struct {
	Aggregations struct {
		ControlIDFilter          AggregationResult `json:"control_id_filter"`
		SeverityFilter           AggregationResult `json:"severity_filter"`
		ConnectorFilter          AggregationResult `json:"connector_filter"`
		ConnectionIDFilter       AggregationResult `json:"connection_id_filter"`
		BenchmarkIDFilter        AggregationResult `json:"benchmark_id_filter"`
		ResourceTypeFilter       AggregationResult `json:"resource_type_filter"`
		ResourceCollectionFilter AggregationResult `json:"resource_collection_filter"`
		ConformanceStatusFilter  AggregationResult `json:"conformance_status_filter"`
		StateActiveFilter        struct {
			DocCountErrorUpperBound int `json:"doc_count_error_upper_bound"`
			SumOtherDocCount        int `json:"sum_other_doc_count"`
			Buckets                 []struct {
				KeyAsString string `json:"key_as_string"`
				DocCount    int    `json:"doc_count"`
			} `json:"buckets"`
		} `json:"state_active_filter"`
	} `json:"aggregations"`
}

func FindingsFiltersQuery(ctx context.Context, logger *zap.Logger, client kaytu.Client,
	resourceIDs []string, connector []source.Type, connectionID []string, notConnectionID []string,
	resourceTypes []string, benchmarkID []string, controlID []string, severity []types.FindingSeverity,
	lastTransitionFrom *time.Time, lastTransitionTo *time.Time,
	evaluatedAtFrom *time.Time, evaluatedAtTo *time.Time,
	stateActive []bool, conformanceStatuses []types.ConformanceStatus,
) (*FindingFiltersAggregationResponse, error) {
	idx := types.FindingsIndex

	var filters []kaytu.BoolFilter
	if len(resourceIDs) > 0 {
		filters = append(filters, kaytu.NewTermsFilter("resourceID", resourceIDs))
	}
	if len(resourceTypes) > 0 {
		filters = append(filters, kaytu.NewTermsFilter("resourceType", resourceTypes))
	}
	if len(benchmarkID) > 0 {
		filters = append(filters, kaytu.NewTermsFilter("parentBenchmarks", benchmarkID))
	}
	if len(controlID) > 0 {
		filters = append(filters, kaytu.NewTermsFilter("controlID", controlID))
	}
	if len(severity) > 0 {
		strSeverity := make([]string, 0)
		for _, s := range severity {
			strSeverity = append(strSeverity, string(s))
		}
		filters = append(filters, kaytu.NewTermsFilter("severity", strSeverity))
	}
	if len(conformanceStatuses) > 0 {
		strConformanceStatus := make([]string, 0)
		for _, cr := range conformanceStatuses {
			strConformanceStatus = append(strConformanceStatus, string(cr))
		}
		filters = append(filters, kaytu.NewTermsFilter("conformanceStatus", strConformanceStatus))
	}
	if len(connectionID) > 0 {
		filters = append(filters, kaytu.NewTermsFilter("connectionID", connectionID))
	}
	if len(notConnectionID) > 0 {
		filters = append(filters, kaytu.NewBoolMustNotFilter(kaytu.NewTermsFilter("connectionID", notConnectionID)))
	}
	if len(connector) > 0 {
		var connectors []string
		for _, p := range connector {
			connectors = append(connectors, p.String())
		}
		filters = append(filters, kaytu.NewTermsFilter("connector", connectors))
	}
	if len(stateActive) > 0 {
		strStateActive := make([]string, 0)
		for _, s := range stateActive {
			strStateActive = append(strStateActive, fmt.Sprintf("%v", s))
		}
		filters = append(filters, kaytu.NewTermsFilter("stateActive", strStateActive))
	}
	if lastTransitionFrom != nil && lastTransitionTo != nil {
		filters = append(filters, kaytu.NewRangeFilter("lastTransition",
			"", fmt.Sprintf("%d", lastTransitionFrom.UnixMilli()),
			"", fmt.Sprintf("%d", lastTransitionTo.UnixMilli())))
	} else if lastTransitionFrom != nil {
		filters = append(filters, kaytu.NewRangeFilter("lastTransition",
			"", fmt.Sprintf("%d", lastTransitionFrom.UnixMilli()),
			"", ""))
	} else if lastTransitionTo != nil {
		filters = append(filters, kaytu.NewRangeFilter("lastTransition",
			"", "",
			"", fmt.Sprintf("%d", lastTransitionTo.UnixMilli())))
	}
	if evaluatedAtFrom != nil && evaluatedAtTo != nil {
		filters = append(filters, kaytu.NewRangeFilter("evaluatedAt",
			"", fmt.Sprintf("%d", evaluatedAtFrom.UnixMilli()),
			"", fmt.Sprintf("%d", evaluatedAtTo.UnixMilli())))
	} else if evaluatedAtFrom != nil {
		filters = append(filters, kaytu.NewRangeFilter("evaluatedAt",
			"", fmt.Sprintf("%d", evaluatedAtFrom.UnixMilli()),
			"", ""))
	} else if evaluatedAtTo != nil {
		filters = append(filters, kaytu.NewRangeFilter("evaluatedAt",
			"", "",
			"", fmt.Sprintf("%d", evaluatedAtTo.UnixMilli())))
	}

	root := map[string]any{}
	root["size"] = 0

	aggs := map[string]any{
		"connector_filter":           map[string]any{"terms": map[string]any{"field": "connector", "size": 1000}},
		"resource_type_filter":       map[string]any{"terms": map[string]any{"field": "resourceType", "size": 1000}},
		"connection_id_filter":       map[string]any{"terms": map[string]any{"field": "connectionID", "size": 1000}},
		"resource_collection_filter": map[string]any{"terms": map[string]any{"field": "resourceCollection", "size": 1000}},
		"benchmark_id_filter":        map[string]any{"terms": map[string]any{"field": "benchmarkID", "size": 1000}},
		"control_id_filter":          map[string]any{"terms": map[string]any{"field": "controlID", "size": 1000}},
		"severity_filter":            map[string]any{"terms": map[string]any{"field": "severity", "size": 1000}},
		"conformance_status_filter":  map[string]any{"terms": map[string]any{"field": "conformanceStatus", "size": 1000}},
		"state_active_filter":        map[string]any{"terms": map[string]any{"field": "stateActive", "size": 1000}},
	}
	root["aggs"] = aggs

	if len(filters) > 0 {
		root["query"] = map[string]any{
			"bool": map[string]any{
				"filter": filters,
			},
		}
	}

	queryBytes, err := json.Marshal(root)
	if err != nil {
		logger.Error("FindingsFiltersQuery", zap.Error(err), zap.String("query", string(queryBytes)), zap.String("index", idx))
		return nil, err
	}

	logger.Info("FindingsFiltersQuery", zap.String("query", string(queryBytes)), zap.String("index", idx))

	var resp FindingFiltersAggregationResponse
	err = client.Search(ctx, idx, string(queryBytes), &resp)
	if err != nil {
		logger.Error("FindingsFiltersQuery", zap.Error(err), zap.String("query", string(queryBytes)), zap.String("index", idx))
		return nil, err
	}

	return &resp, nil
}

type FindingKPIResponse struct {
	Hits struct {
		Total struct {
			Value int64 `json:"value"`
		} `json:"total"`
	} `json:"hits"`
	Aggregations struct {
		ResourceCount struct {
			Value int64 `json:"value"`
		} `json:"resource_count"`
		ControlCount struct {
			Value int64 `json:"value"`
		} `json:"control_count"`
		ConnectionCount struct {
			Value int64 `json:"value"`
		} `json:"connection_count"`
	} `json:"aggregations"`
}

func FindingKPIQuery(ctx context.Context, logger *zap.Logger, client kaytu.Client) (*FindingKPIResponse, error) {
	root := make(map[string]any)
	root["size"] = 0
	root["track_total_hits"] = true

	filters := make([]map[string]any, 0)
	filters = append(filters, map[string]any{
		"terms": map[string]any{
			"conformanceStatus": types.GetFailedConformanceStatuses(),
		},
	})
	root["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}

	root["aggs"] = map[string]any{
		"resource_count": map[string]any{
			"cardinality": map[string]any{
				"field": "kaytuResourceID",
			},
		},
		"control_count": map[string]any{
			"cardinality": map[string]any{
				"field": "controlID",
			},
		},
		"connection_count": map[string]any{
			"cardinality": map[string]any{
				"field": "connectionID",
			},
		},
	}

	queryBytes, err := json.Marshal(root)
	if err != nil {
		return nil, err
	}

	logger.Info("FindingKPIQuery", zap.String("query", string(queryBytes)))
	var resp FindingKPIResponse
	err = client.SearchWithTrackTotalHits(ctx, types.FindingsIndex, string(queryBytes), nil, &resp, true)
	if err != nil {
		logger.Error("FindingKPIQuery", zap.Error(err), zap.String("query", string(queryBytes)))
		return nil, err
	}
	return &resp, err
}

type FindingsTopFieldResponse struct {
	Aggregations struct {
		FieldFilter struct {
			DocCountErrorUpperBound int `json:"doc_count_error_upper_bound"`
			SumOtherDocCount        int `json:"sum_other_doc_count"`
			Buckets                 []struct {
				Key      string `json:"key"`
				DocCount int    `json:"doc_count"`
			} `json:"buckets"`
		} `json:"field_filter"`
		BucketCount struct {
			Value int `json:"value"`
		} `json:"bucket_count"`
	} `json:"aggregations"`
}

func FindingsTopFieldQuery(ctx context.Context, logger *zap.Logger, client kaytu.Client,
	field string, connectors []source.Type, resourceTypeID []string, connectionIDs []string, notConnectionIDs []string, jobIDs []string,
	benchmarkID []string, controlID []string, severity []types.FindingSeverity, conformanceStatuses []types.ConformanceStatus, stateActives []bool,
	size int) (*FindingsTopFieldResponse, error) {
	filters := make([]kaytu.BoolFilter, 0)

	idx := types.FindingsIndex
	if len(benchmarkID) > 0 {
		filters = append(filters, kaytu.NewTermsFilter("benchmarkID", benchmarkID))
	}

	if len(controlID) > 0 {
		filters = append(filters, kaytu.NewTermsFilter("controlID", controlID))
	}

	if len(conformanceStatuses) > 0 {
		cfStrs := make([]string, 0, len(conformanceStatuses))
		for _, cf := range conformanceStatuses {
			cfStrs = append(cfStrs, string(cf))
		}
		filters = append(filters, kaytu.NewTermsFilter("conformanceStatus", cfStrs))
	}

	if len(severity) > 0 {
		sevStrs := make([]string, 0, len(severity))
		for _, s := range severity {
			sevStrs = append(sevStrs, string(s))
		}
		filters = append(filters, kaytu.NewTermsFilter("severity", sevStrs))
	}

	if len(connectionIDs) > 0 {
		filters = append(filters, kaytu.NewTermsFilter("connectionID", connectionIDs))
	}

	if len(jobIDs) > 0 {
		filters = append(filters, kaytu.NewTermsFilter("parentComplianceJobID", jobIDs))
	}

	if len(notConnectionIDs) > 0 {
		filters = append(filters, kaytu.NewBoolMustNotFilter(kaytu.NewTermsFilter("connectionID", notConnectionIDs)))
	}

	if len(resourceTypeID) > 0 {
		filters = append(filters, kaytu.NewTermsFilter("resourceType", resourceTypeID))
	}

	if len(connectors) > 0 {
		var connectorsStr []string
		for _, c := range connectors {
			connectorsStr = append(connectorsStr, c.String())
		}
		filters = append(filters, kaytu.NewTermsFilter("connector", connectorsStr))
	}

	if len(stateActives) > 0 {
		strStateActive := make([]string, 0)
		for _, s := range stateActives {
			strStateActive = append(strStateActive, fmt.Sprintf("%v", s))
		}
		filters = append(filters, kaytu.NewTermsFilter("stateActive", strStateActive))
	} else {
		filters = append(filters, kaytu.NewTermFilter("stateActive", "true"))
	}

	root := map[string]any{}
	root["size"] = 0

	root["aggs"] = map[string]any{
		"field_filter": map[string]any{
			"terms": map[string]any{
				"field": field,
				"size":  size,
			},
		},
		"bucket_count": map[string]any{
			"cardinality": map[string]any{
				"field": field,
			},
		},
	}

	if len(filters) > 0 {
		root["query"] = map[string]any{
			"bool": map[string]any{
				"filter": filters,
			},
		}
	}

	queryBytes, err := json.Marshal(root)
	if err != nil {
		return nil, err
	}

	logger.Info("FindingsTopFieldQuery", zap.String("query", string(queryBytes)), zap.String("index", idx))
	var resp FindingsTopFieldResponse
	err = client.Search(ctx, idx, string(queryBytes), &resp)
	return &resp, err
}

type ResourceTypesFindingsSummaryResponse struct {
	Aggregations struct {
		Summaries struct {
			Buckets []struct {
				Key      string `json:"key"`
				DocCount int    `json:"doc_count"`
				Severity struct {
					Buckets []struct {
						Key      string `json:"key"`
						DocCount int    `json:"doc_count"`
					} `json:"buckets"`
				} `json:"severity"`
				ConformanceStatus struct {
					Buckets []struct {
						Key      string `json:"key"`
						DocCount int    `json:"doc_count"`
					} `json:"buckets"`
				} `json:"conformanceStatus"`
			} `json:"buckets"`
		} `json:"summaries"`
	} `json:"aggregations"`
}

func ResourceTypesFindingsSummary(ctx context.Context, logger *zap.Logger, client kaytu.Client,
	connectionIDs []string, benchmarkID string) (*ResourceTypesFindingsSummaryResponse, error) {
	var filters []map[string]any

	filters = append(filters, map[string]any{
		"term": map[string]any{
			"parentBenchmarks": benchmarkID,
		},
	})

	if len(connectionIDs) != 0 {
		filters = append(filters, map[string]any{
			"term": map[string]any{
				"connectionID": connectionIDs,
			},
		})
	}

	filters = append(filters, map[string]any{
		"term": map[string]any{
			"stateActive": true,
		},
	})

	request := map[string]any{
		"aggs": map[string]any{
			"summaries": map[string]any{
				"terms": map[string]any{
					"field": "resourceType",
				},
				"aggs": map[string]any{
					"severity": map[string]any{
						"terms": map[string]any{
							"field": "severity",
							"size":  1000,
						},
					},
					"conformanceStatus": map[string]any{
						"terms": map[string]any{
							"field": "conformanceStatus",
							"size":  1000,
						},
					},
				},
			},
		},
		"query": map[string]any{
			"bool": map[string]any{
				"filter": filters,
			},
		},
		"size": 0,
	}

	queryBytes, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	logger.Info("ResourceTypesFindingsSummary", zap.String("query", string(queryBytes)))
	var resp ResourceTypesFindingsSummaryResponse
	err = client.Search(ctx, types.FindingsIndex, string(queryBytes), &resp)
	return &resp, err
}

type FindingsFieldCountByControlResponse struct {
	Aggregations struct {
		ControlCount struct {
			DocCountErrorUpperBound int `json:"doc_count_error_upper_bound"`
			SumOtherDocCount        int `json:"sum_other_doc_count"`
			Buckets                 []struct {
				Key                 string `json:"key"`
				DocCount            int    `json:"doc_count"`
				ConformanceStatuses struct {
					DocCountErrorUpperBound int `json:"doc_count_error_upper_bound"`
					SumOtherDocCount        int `json:"sum_other_doc_count"`
					Buckets                 []struct {
						Key        string `json:"key"`
						DocCount   int    `json:"doc_count"`
						FieldCount struct {
							Value int `json:"value"`
						} `json:"field_count"`
					} `json:"buckets"`
				} `json:"conformanceStatus"`
			} `json:"buckets"`
		} `json:"control_count"`
	} `json:"aggregations"`
}

func FindingsFieldCountByControl(ctx context.Context, logger *zap.Logger, client kaytu.Client,
	field string, connectors []source.Type, resourceTypeID []string, connectionIDs []string, benchmarkID []string, controlID []string,
	severity []types.FindingSeverity, conformanceStatuses []types.ConformanceStatus) (*FindingsFieldCountByControlResponse, error) {
	terms := make(map[string]any)
	idx := types.FindingsIndex
	if len(benchmarkID) > 0 {
		terms["benchmarkID"] = benchmarkID
	}

	if len(controlID) > 0 {
		terms["controlID"] = controlID
	}

	if len(severity) > 0 {
		terms["severity"] = severity
	}

	if len(conformanceStatuses) > 0 {
		terms["conformanceStatus"] = conformanceStatuses
	}

	if len(connectionIDs) > 0 {
		terms["connectionID"] = connectionIDs
	}

	if len(resourceTypeID) > 0 {
		terms["resourceType"] = resourceTypeID
	}

	if len(connectors) > 0 {
		terms["connector"] = connectors
	}

	terms["stateActive"] = []bool{true}

	root := map[string]any{}
	root["size"] = 0

	root["aggs"] = map[string]any{
		"control_count": map[string]any{
			"terms": map[string]any{
				"field": "controlID",
			},
			"aggs": map[string]any{
				"conformanceStatus": map[string]any{
					"terms": map[string]any{
						"field": "conformanceStatus",
					},
					"aggs": map[string]any{
						"field_count": map[string]any{
							"cardinality": map[string]any{
								"field": field,
							},
						},
					},
				},
			},
		},
	}

	boolQuery := make(map[string]any)
	if terms != nil && len(terms) > 0 {
		var filters []map[string]any
		for k, vs := range terms {
			filters = append(filters, map[string]any{
				"terms": map[string]any{
					k: vs,
				},
			})
		}

		boolQuery["filter"] = filters
	}
	if len(boolQuery) > 0 {
		root["query"] = map[string]any{
			"bool": boolQuery,
		}
	}

	queryBytes, err := json.Marshal(root)
	if err != nil {
		return nil, err
	}

	logger.Info("FindingsFieldCountByControl", zap.String("query", string(queryBytes)), zap.String("index", idx))
	var resp FindingsFieldCountByControlResponse
	err = client.Search(ctx, idx, string(queryBytes), &resp)
	return &resp, err
}

type FindingsConformanceStatusCountByControlPerConnectionResponse struct {
	Aggregations struct {
		ConnectionGroup struct {
			DocCountErrorUpperBound int `json:"doc_count_error_upper_bound"`
			SumOtherDocCount        int `json:"sum_other_doc_count"`
			Buckets                 []struct {
				Key          string `json:"key"`
				ControlCount struct {
					DocCountErrorUpperBound int `json:"doc_count_error_upper_bound"`
					SumOtherDocCount        int `json:"sum_other_doc_count"`
					Buckets                 []struct {
						Key                 string `json:"key"`
						DocCount            int    `json:"doc_count"`
						ConformanceStatuses struct {
							Key     string `json:"key"`
							Buckets []struct {
								Key      string `json:"key"`
								DocCount int    `json:"doc_count"`
							} `json:"buckets"`
						} `json:"conformanceStatus"`
					} `json:"buckets"`
				} `json:"control_count"`
			} `json:"buckets"`
		} `json:"connection_group"`
	} `json:"aggregations"`
}

func FindingsConformanceStatusCountByControlPerConnection(ctx context.Context, logger *zap.Logger, client kaytu.Client,
	connectors []source.Type, resourceTypeID []string, connectionIDs []string, benchmarkID []string, controlID []string,
	severity []types.FindingSeverity, conformanceStatuses []types.ConformanceStatus) (*FindingsConformanceStatusCountByControlPerConnectionResponse, error) {
	terms := make(map[string]any)
	idx := types.FindingsIndex
	if len(benchmarkID) > 0 {
		terms["benchmarkID"] = benchmarkID
	}

	if len(controlID) > 0 {
		terms["controlID"] = controlID
	}

	if len(severity) > 0 {
		terms["severity"] = severity
	}

	if len(conformanceStatuses) > 0 {
		terms["conformanceStatus"] = conformanceStatuses
	}

	if len(connectionIDs) > 0 {
		terms["connectionID"] = connectionIDs
	}

	if len(resourceTypeID) > 0 {
		terms["resourceType"] = resourceTypeID
	}

	if len(connectors) > 0 {
		terms["connector"] = connectors
	}

	terms["stateActive"] = []bool{true}

	root := map[string]any{}
	root["size"] = 0

	root["aggs"] = map[string]any{
		"connection_group": map[string]any{
			"terms": map[string]any{
				"field": "connectionID",
				"size":  10000,
			},
			"aggs": map[string]any{
				"control_count": map[string]any{
					"terms": map[string]any{
						"field": "controlID",
						"size":  10000,
					},
					"aggs": map[string]any{
						"conformanceStatus": map[string]any{
							"terms": map[string]any{
								"field": "conformanceStatus",
								"size":  10000,
							},
						},
					},
				},
			},
		},
	}

	boolQuery := make(map[string]any)
	if terms != nil && len(terms) > 0 {
		var filters []map[string]any
		for k, vs := range terms {
			filters = append(filters, map[string]any{
				"terms": map[string]any{
					k: vs,
				},
			})
		}

		boolQuery["filter"] = filters
	}
	if len(boolQuery) > 0 {
		root["query"] = map[string]any{
			"bool": boolQuery,
		}
	}

	queryBytes, err := json.Marshal(root)
	if err != nil {
		return nil, err
	}

	logger.Info("FindingsFieldCountByControl", zap.String("query", string(queryBytes)), zap.String("index", idx))
	var resp FindingsConformanceStatusCountByControlPerConnectionResponse
	err = client.Search(ctx, idx, string(queryBytes), &resp)
	if err != nil {
		logger.Error("FindingsFieldCountByControl", zap.Error(err), zap.String("query", string(queryBytes)), zap.String("index", idx))
		return nil, err
	}
	return &resp, nil
}

type FindingCountPerKaytuResourceIdsResponse struct {
	Aggregations struct {
		KaytuResourceIDGroup struct {
			Buckets []struct {
				Key      string `json:"key"`
				DocCount int    `json:"doc_count"`
			} `json:"buckets"`
		} `json:"kaytu_resource_id_group"`
	} `json:"aggregations"`
}

func FetchFindingCountPerKaytuResourceIds(ctx context.Context, logger *zap.Logger, client kaytu.Client, kaytuResourceIds []string,
	severities []types.FindingSeverity, conformanceStatuses []types.ConformanceStatus,
) (map[string]int, error) {
	var filters []map[string]any

	if len(kaytuResourceIds) == 0 {
		return map[string]int{}, nil
	}

	filters = append(filters, map[string]any{
		"terms": map[string]any{
			"kaytuResourceID": kaytuResourceIds,
		},
	})
	if len(severities) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string]any{
				"severity": severities,
			},
		})
	}
	if len(conformanceStatuses) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string]any{
				"conformanceStatus": conformanceStatuses,
			},
		})
	}
	filters = append(filters, map[string]any{
		"term": map[string]any{
			"stateActive": true,
		},
	})

	request := map[string]any{
		"aggs": map[string]any{
			"kaytu_resource_id_group": map[string]any{
				"terms": map[string]any{
					"field": "kaytuResourceID",
					"size":  len(kaytuResourceIds),
				},
			},
		},
		"query": map[string]any{
			"bool": map[string]any{
				"filter": filters,
			},
		},
		"size": 0,
	}

	queryBytes, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	logger.Info("FetchFindingCountPerKaytuResourceIds", zap.String("query", string(queryBytes)))
	var resp FindingCountPerKaytuResourceIdsResponse
	err = client.Search(ctx, types.FindingsIndex, string(queryBytes), &resp)
	if err != nil {
		return nil, err
	}

	result := make(map[string]int)
	for _, bucket := range resp.Aggregations.KaytuResourceIDGroup.Buckets {
		result[bucket.Key] = bucket.DocCount
	}

	return result, nil
}

type FindingsPerControlForResourceIdResponse struct {
	Aggregations struct {
		ControlIDGroup struct {
			Buckets []struct {
				Key       string `json:"key"`
				HitSelect struct {
					Hits struct {
						Hits []struct {
							Source types.Finding `json:"_source"`
						} `json:"hits"`
					} `json:"hits"`
				} `json:"hit_select"`
			} `json:"buckets"`
		} `json:"control_id_group"`
	} `json:"aggregations"`
}

func FetchFindingsPerControlForResourceId(ctx context.Context, logger *zap.Logger, client kaytu.Client, kaytuResourceId string) ([]types.Finding, error) {
	request := map[string]any{
		"aggs": map[string]any{
			"control_id_group": map[string]any{
				"terms": map[string]any{
					"field": "controlID",
					"size":  10000,
				},
				"aggs": map[string]any{
					"hit_select": map[string]any{
						"top_hits": map[string]any{
							"sort": map[string]any{
								"parentComplianceJobID": "desc",
							},
							"size": 1,
						},
					},
				},
			},
		},
		"query": map[string]any{
			"bool": map[string]any{
				"filter": map[string]any{
					"term": map[string]any{
						"kaytuResourceID": kaytuResourceId,
					},
				},
			},
		},
		"size": 0,
	}

	queryBytes, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}

	logger.Info("FetchFindingsPerControlForResourceId", zap.String("query", string(queryBytes)))
	var resp FindingsPerControlForResourceIdResponse
	err = client.Search(ctx, types.FindingsIndex, string(queryBytes), &resp)
	if err != nil {
		return nil, err
	}

	var findings []types.Finding
	for _, bucket := range resp.Aggregations.ControlIDGroup.Buckets {
		for _, hit := range bucket.HitSelect.Hits.Hits {
			findings = append(findings, hit.Source)
		}
	}

	return findings, nil
}

func FetchFindingByID(ctx context.Context, logger *zap.Logger, client kaytu.Client, findingID string) (*types.Finding, error) {
	query := map[string]any{
		"query": map[string]any{
			"term": map[string]any{
				"es_id": findingID,
			},
		},
		"size": 1,
	}

	queryBytes, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}

	logger.Info("FetchFindingByID", zap.String("query", string(queryBytes)))
	var resp FindingsQueryResponse
	err = client.Search(ctx, types.FindingsIndex, string(queryBytes), &resp)
	if err != nil {
		return nil, err
	}

	if len(resp.Hits.Hits) == 0 {
		return nil, nil
	}

	return &resp.Hits.Hits[0].Source, nil
}

func FindingsQueryV2(ctx context.Context, logger *zap.Logger, client kaytu.Client, resourceIDs []string, notResourceIDs []string,
	provider []source.Type, connectionID []string, notConnectionID []string, resourceTypes []string, notResourceTypes []string,
	benchmarkID []string, notBenchmarkID []string, controlID []string, notControlID []string, severity []types.FindingSeverity,
	notSeverity []types.FindingSeverity, lastTransitionFrom *time.Time, lastTransitionTo *time.Time, notLastTransitionFrom *time.Time,
	notLastTransitionTo *time.Time, evaluatedAtFrom *time.Time, evaluatedAtTo *time.Time, stateActive []bool,
	conformanceStatuses []types.ConformanceStatus, sorts []api.FindingsSortV2, pageSizeLimit int, searchAfter []any) ([]FindingsQueryHit, int64, error) {
	idx := types.FindingsIndex

	requestSort := make([]map[string]any, 0, len(sorts)+1)
	for _, sort := range sorts {
		switch {
		case sort.ResourceType != nil:
			requestSort = append(requestSort, map[string]any{
				"resourceType": *sort.ResourceType,
			})
		case sort.BenchmarkID != nil:
			requestSort = append(requestSort, map[string]any{
				"benchmarkID": *sort.BenchmarkID,
			})
		case sort.ControlID != nil:
			requestSort = append(requestSort, map[string]any{
				"controlID": *sort.ControlID,
			})
		case sort.Severity != nil:
			scriptSource :=
				`if (params['_source']['severity'] == 'critical') {
					return 5
				} else if (params['_source']['severity'] == 'high') {
					return 4
				} else if (params['_source']['severity'] == 'medium') {
					return 3
				} else if (params['_source']['severity'] == 'low') {
					return 2
				} else if (params['_source']['severity'] == 'none') {
					return 1
				} else {
					return 1
				}`
			requestSort = append(requestSort, map[string]any{
				"_script": map[string]any{
					"type": "number",
					"script": map[string]any{
						"lang":   "painless",
						"source": scriptSource,
					},
					"order": *sort.Severity,
				},
			})
		case sort.ConformanceStatus != nil:
			scriptSource :=
				`if (params['_source']['conformanceStatus'] == 'alarm') {
					return 5
				} else if (params['_source']['conformanceStatus'] == 'error') {
					return 4
				} else if (params['_source']['conformanceStatus'] == 'info') {
					return 3
				} else if (params['_source']['conformanceStatus'] == 'skip') {
					return 2
				} else if (params['_source']['conformanceStatus'] == 'ok') {
					return 1
				} else {
					return 1
				}`
			requestSort = append(requestSort, map[string]any{
				"_script": map[string]any{
					"type": "number",
					"script": map[string]any{
						"lang":   "painless",
						"source": scriptSource,
					},
					"order": *sort.ConformanceStatus,
				},
			})
		case sort.LastUpdated != nil:
			requestSort = append(requestSort, map[string]any{
				"lastTransition": *sort.LastUpdated,
			})
		}
	}
	requestSort = append(requestSort, map[string]any{
		"_id": "asc",
	})

	var filters []kaytu.BoolFilter
	if len(resourceIDs) > 0 {
		filters = append(filters, kaytu.NewTermsFilter("resourceID", resourceIDs))
	}
	if len(notResourceIDs) > 0 {
		filters = append(filters, kaytu.NewBoolMustNotFilter(kaytu.NewTermsFilter("resourceID", notResourceIDs)))
	}
	if len(resourceTypes) > 0 {
		filters = append(filters, kaytu.NewTermsFilter("resourceType", resourceTypes))
	}
	if len(notResourceTypes) > 0 {
		filters = append(filters, kaytu.NewBoolMustNotFilter(kaytu.NewTermsFilter("resourceType", notResourceTypes)))
	}
	if len(benchmarkID) > 0 {
		filters = append(filters, kaytu.NewTermsFilter("parentBenchmarks", benchmarkID))
	}
	if len(notBenchmarkID) > 0 {
		filters = append(filters, kaytu.NewBoolMustNotFilter(kaytu.NewTermsFilter("parentBenchmarks", notBenchmarkID)))
	}
	if len(controlID) > 0 {
		filters = append(filters, kaytu.NewTermsFilter("controlID", controlID))
	}
	if len(notControlID) > 0 {
		filters = append(filters, kaytu.NewBoolMustNotFilter(kaytu.NewTermsFilter("controlID", notControlID)))
	}
	if len(severity) > 0 {
		strSeverity := make([]string, 0)
		for _, s := range severity {
			strSeverity = append(strSeverity, string(s))
		}
		filters = append(filters, kaytu.NewTermsFilter("severity", strSeverity))
	}
	if len(notSeverity) > 0 {
		strSeverity := make([]string, 0)
		for _, s := range notSeverity {
			strSeverity = append(strSeverity, string(s))
		}
		filters = append(filters, kaytu.NewBoolMustNotFilter(kaytu.NewTermsFilter("severity", strSeverity)))
	}
	if len(conformanceStatuses) > 0 {
		strConformanceStatus := make([]string, 0)
		for _, cr := range conformanceStatuses {
			strConformanceStatus = append(strConformanceStatus, string(cr))
		}
		filters = append(filters, kaytu.NewTermsFilter("conformanceStatus", strConformanceStatus))
	}
	if len(connectionID) > 0 {
		filters = append(filters, kaytu.NewTermsFilter("connectionID", connectionID))
	}
	if len(notConnectionID) > 0 {
		filters = append(filters, kaytu.NewBoolMustNotFilter(kaytu.NewTermsFilter("connectionID", notConnectionID)))
	}
	if len(provider) > 0 {
		var connectors []string
		for _, p := range provider {
			connectors = append(connectors, p.String())
		}
		filters = append(filters, kaytu.NewTermsFilter("connector", connectors))
	}
	if len(stateActive) > 0 {
		strStateActive := make([]string, 0)
		for _, s := range stateActive {
			strStateActive = append(strStateActive, fmt.Sprintf("%v", s))
		}
		filters = append(filters, kaytu.NewTermsFilter("stateActive", strStateActive))
	}
	if lastTransitionFrom != nil && lastTransitionTo != nil {
		filters = append(filters, kaytu.NewRangeFilter("lastTransition",
			"", fmt.Sprintf("%d", lastTransitionFrom.UnixMilli()),
			"", fmt.Sprintf("%d", lastTransitionTo.UnixMilli())))
	} else if lastTransitionFrom != nil {
		filters = append(filters, kaytu.NewRangeFilter("lastTransition",
			"", fmt.Sprintf("%d", lastTransitionFrom.UnixMilli()),
			"", ""))
	} else if lastTransitionTo != nil {
		filters = append(filters, kaytu.NewRangeFilter("lastTransition",
			"", "",
			"", fmt.Sprintf("%d", lastTransitionTo.UnixMilli())))
	}
	if notLastTransitionFrom != nil && notLastTransitionTo != nil {
		filters = append(filters, kaytu.NewBoolMustNotFilter(kaytu.NewRangeFilter("lastTransition",
			"", fmt.Sprintf("%d", notLastTransitionFrom.UnixMilli()),
			"", fmt.Sprintf("%d", notLastTransitionTo.UnixMilli()))))
	} else if notLastTransitionFrom != nil {
		filters = append(filters, kaytu.NewBoolMustNotFilter(kaytu.NewRangeFilter("lastTransition",
			"", fmt.Sprintf("%d", notLastTransitionFrom.UnixMilli()),
			"", "")))
	} else if notLastTransitionTo != nil {
		filters = append(filters, kaytu.NewBoolMustNotFilter(kaytu.NewRangeFilter("lastTransition",
			"", "",
			"", fmt.Sprintf("%d", notLastTransitionTo.UnixMilli()))))
	}
	if evaluatedAtFrom != nil && evaluatedAtTo != nil {
		filters = append(filters, kaytu.NewRangeFilter("evaluatedAt",
			"", fmt.Sprintf("%d", evaluatedAtFrom.UnixMilli()),
			"", fmt.Sprintf("%d", evaluatedAtTo.UnixMilli())))
	} else if evaluatedAtFrom != nil {
		filters = append(filters, kaytu.NewRangeFilter("evaluatedAt",
			"", fmt.Sprintf("%d", evaluatedAtFrom.UnixMilli()),
			"", ""))
	} else if evaluatedAtTo != nil {
		filters = append(filters, kaytu.NewRangeFilter("evaluatedAt",
			"", "",
			"", fmt.Sprintf("%d", evaluatedAtTo.UnixMilli())))
	}

	query := make(map[string]any)
	if len(filters) > 0 {
		query["query"] = map[string]any{
			"bool": map[string]any{
				"filter": filters,
			},
		}
	}
	query["sort"] = requestSort
	if len(searchAfter) > 0 {
		query["search_after"] = searchAfter
	}
	if pageSizeLimit == 0 {
		pageSizeLimit = 1000
	}
	query["size"] = pageSizeLimit
	queryJson, err := json.Marshal(query)
	if err != nil {
		return nil, 0, err
	}

	logger.Info("FindingsQuery", zap.String("query", string(queryJson)), zap.String("index", idx))

	var response FindingsQueryResponse
	err = client.SearchWithTrackTotalHits(ctx, idx, string(queryJson), nil, &response, true)
	if err != nil {
		return nil, 0, err
	}

	return response.Hits.Hits, response.Hits.Total.Value, err
}
