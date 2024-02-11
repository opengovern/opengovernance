package es

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"go.uber.org/zap"
	"time"
)

type FindingEventsQueryHit struct {
	ID      string             `json:"_id"`
	Score   float64            `json:"_score"`
	Index   string             `json:"_index"`
	Type    string             `json:"_type"`
	Version int64              `json:"_version,omitempty"`
	Source  types.FindingEvent `json:"_source"`
	Sort    []any              `json:"sort"`
}

type FindingEventsQueryResponse struct {
	Hits struct {
		Total kaytu.SearchTotal       `json:"total"`
		Hits  []FindingEventsQueryHit `json:"hits"`
	} `json:"hits"`
	PitID string `json:"pit_id"`
}

type FetchFindingEventsByFindingIDResponse struct {
	Hits struct {
		Hits []FindingEventsQueryHit `json:"hits"`
	} `json:"hits"`
}

func FetchFindingEventsByFindingIDs(logger *zap.Logger, client kaytu.Client, findingID []string) ([]types.FindingEvent, error) {
	request := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"filter": []any{
					map[string]any{
						"terms": map[string][]string{
							"findingEsID": findingID,
						},
					},
				},
			},
		},
		"sort": map[string]any{
			"evaluatedAt": "desc",
		},
		"size": 10000,
	}

	jsonReq, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	logger.Info("Fetching finding events", zap.String("request", string(jsonReq)), zap.String("index", types.FindingEventsIndex))

	var resp FetchFindingEventsByFindingIDResponse
	err = client.Search(context.Background(), types.FindingEventsIndex, string(jsonReq), &resp)
	if err != nil {
		logger.Error("Failed to fetch finding events", zap.Error(err), zap.String("request", string(jsonReq)), zap.String("index", types.FindingEventsIndex))
		return nil, err
	}
	result := make([]types.FindingEvent, 0, len(resp.Hits.Hits))
	for _, hit := range resp.Hits.Hits {
		result = append(result, hit.Source)
	}
	return result, nil
}

func FindingEventsQuery(logger *zap.Logger, client kaytu.Client,
	findingIDs []string, kaytuResourceIDs []string,
	provider []source.Type, connectionID []string, notConnectionID []string,
	resourceTypes []string,
	benchmarkID []string, controlID []string, severity []types.FindingSeverity,
	evaluatedAtFrom *time.Time, evaluatedAtTo *time.Time,
	stateActive []bool, conformanceStatuses []types.ConformanceStatus,
	sorts []api.FindingEventsSort, pageSizeLimit int, searchAfter []any) ([]FindingEventsQueryHit, int64, error) {
	idx := types.FindingEventsIndex

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

		case sort.ResourceType != nil:
			requestSort = append(requestSort, map[string]any{
				"resourceType": *sort.ResourceType,
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
	if len(findingIDs) > 0 {
		filters = append(filters, kaytu.NewTermsFilter("findingEsID", findingIDs))
	}
	if len(kaytuResourceIDs) > 0 {
		filters = append(filters, kaytu.NewTermsFilter("kaytuResourceID", kaytuResourceIDs))
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

	logger.Info("FindingEventsQuery", zap.String("query", string(queryJson)), zap.String("index", idx))

	var response FindingEventsQueryResponse
	err = client.SearchWithTrackTotalHits(context.Background(), idx, string(queryJson), nil, &response, true)
	if err != nil {
		return nil, 0, err
	}

	return response.Hits.Hits, response.Hits.Total.Value, err
}

type FindingEventFiltersAggregationResponse struct {
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

func FindingEventsFiltersQuery(logger *zap.Logger, client kaytu.Client,
	findingIDs []string, kaytuResourceIDs []string, connector []source.Type, connectionID []string, notConnectionID []string,
	resourceTypes []string, benchmarkID []string, controlID []string, severity []types.FindingSeverity,
	evaluatedAtFrom *time.Time, evaluatedAtTo *time.Time,
	stateActive []bool, conformanceStatuses []types.ConformanceStatus,
) (*FindingEventFiltersAggregationResponse, error) {
	idx := types.FindingEventsIndex

	var filters []kaytu.BoolFilter
	if len(findingIDs) > 0 {
		filters = append(filters, kaytu.NewTermsFilter("findingEsID", findingIDs))
	}
	if len(kaytuResourceIDs) > 0 {
		filters = append(filters, kaytu.NewTermsFilter("kaytuResourceID", kaytuResourceIDs))
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
		logger.Error("FindingEventsFiltersQuery", zap.Error(err), zap.String("query", string(queryBytes)), zap.String("index", idx))
		return nil, err
	}

	logger.Info("FindingEventsFiltersQuery", zap.String("query", string(queryBytes)), zap.String("index", idx))

	var resp FindingEventFiltersAggregationResponse
	err = client.Search(context.Background(), idx, string(queryBytes), &resp)
	if err != nil {
		logger.Error("FindingEventsFiltersQuery", zap.Error(err), zap.String("query", string(queryBytes)), zap.String("index", idx))
		return nil, err
	}

	return &resp, nil
}

func FetchFindingEventByID(logger *zap.Logger, client kaytu.Client, findingID string) (*types.FindingEvent, error) {
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
	var resp FindingEventsQueryResponse
	err = client.Search(context.Background(), types.FindingEventsIndex, string(queryBytes), &resp)
	if err != nil {
		return nil, err
	}

	if len(resp.Hits.Hits) == 0 {
		return nil, nil
	}

	return &resp.Hits.Hits[0].Source, nil
}

type FindingEventsCountResponse struct {
	Hits struct {
		Total kaytu.SearchTotal `json:"total"`
	} `json:"hits"`
	PitID string `json:"pit_id"`
}

func FindingEventsCount(client kaytu.Client, benchmarkIDs []string, conformanceStatuses []types.ConformanceStatus, stateActives []bool, startTime, endTime *time.Time) (int64, error) {
	idx := types.FindingEventsIndex

	filters := make([]map[string]any, 0)
	if len(conformanceStatuses) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string]any{
				"conformanceStatus": conformanceStatuses,
			},
		})
	}
	if len(stateActives) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string]any{
				"stateActive": stateActives,
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
	if len(benchmarkIDs) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{
				"benchmarkID": benchmarkIDs,
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
	err = client.SearchWithTrackTotalHits(context.Background(), idx, string(queryJson), nil, &response, true)
	if err != nil {
		return 0, err
	}

	return response.Hits.Total.Value, err
}
