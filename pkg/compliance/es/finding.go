package es

import (
	"context"
	"encoding/json"
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"go.uber.org/zap"
	"sort"
	"strings"

	"github.com/kaytu-io/kaytu-util/pkg/source"

	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
)

type FindingsQueryResponse struct {
	Hits  FindingsQueryHits `json:"hits"`
	PitID string            `json:"pit_id"`
}
type FindingsQueryHits struct {
	Total kaytu.SearchTotal  `json:"total"`
	Hits  []FindingsQueryHit `json:"hits"`
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

func NewFindingPaginator(client kaytu.Client, idx string, filters []kaytu.BoolFilter, limit *int64) (FindingPaginator, error) {
	paginator, err := kaytu.NewPaginator(client.ES(), idx, filters, limit)
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

func FindingsQuery(logger *zap.Logger, client kaytu.Client, resourceIDs []string,
	provider []source.Type, connectionID []string,
	resourceTypes []string, resourceCollections []string,
	benchmarkID []string, controlID []string, severity []string, conformanceStatuses []types.ConformanceStatus,
	sorts map[string]string, pageSizeLimit int, searchAfter []any) ([]FindingsQueryHit, int64, error) {
	idx := types.FindingsIndex

	filteredSortMap := make(map[string]string)
	for sortField, sortDirection := range sorts {
		key := ""
		switch sortField {
		case "controlTitle":
			key = "controlID"
		case "providerConnectionID", "providerConnectionName":
			key = "connectionID"
		default:
			key = sortField
		}
		filteredSortMap[key] = sortDirection
	}
	sortMapArray := make([]map[string]string, 0)
	for k, v := range filteredSortMap {
		sortMapArray = append(sortMapArray, map[string]string{k: v})
	}
	sort.Slice(sortMapArray, func(i, j int) bool {
		for k := range sortMapArray[i] {
			for l := range sortMapArray[j] {
				return k < l
			}
		}
		return false
	})
	sortMapArray = append(sortMapArray, map[string]string{"_id": "asc"})

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
		filters = append(filters, kaytu.NewTermsFilter("severity", severity))
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
	if len(provider) > 0 {
		var connectors []string
		for _, p := range provider {
			connectors = append(connectors, p.String())
		}
		filters = append(filters, kaytu.NewTermsFilter("connector", connectors))
	}
	if len(resourceCollections) > 0 {
		idx = types.ResourceCollectionsFindingsIndex
		filters = append(filters, kaytu.NewTermsFilter("resourceCollection", resourceCollections))
	}

	isStack := false
	if len(connectionID) > 0 {
		if strings.HasPrefix(connectionID[0], "stack-") {
			isStack = true
		}
	}
	if isStack {
		idx = types.StackFindingsIndex
	}

	query := make(map[string]any)
	query["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}
	query["sort"] = sortMapArray
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
	err = client.SearchWithTrackTotalHits(context.Background(), idx, string(queryJson), nil, &response, true)
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

func FindingsCount(client kaytu.Client) (int64, error) {
	idx := types.FindingsIndex

	query := make(map[string]any)
	query["size"] = 0
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

type AggregationResult struct {
	DocCountErrorUpperBound int      `json:"doc_count_error_upper_bound"`
	SumOtherDocCount        int      `json:"sum_other_doc_count"`
	Buckets                 []Bucket `json:"buckets"`
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
	} `json:"aggregations"`
}

func FindingsFiltersQuery(logger *zap.Logger, client kaytu.Client,
	resourceIDs []string, connector []source.Type, connectionID []string, resourceCollections []string,
	benchmarkID []string, controlID []string, severity []string, conformanceStatuses []types.ConformanceStatus,
) (*FindingFiltersAggregationResponse, error) {
	idx := types.FindingsIndex
	terms := make(map[string]any)

	if len(resourceIDs) > 0 {
		terms["resourceID"] = resourceIDs
	}
	if len(connector) > 0 {
		terms["connector"] = connector
	}
	if len(connectionID) > 0 {
		terms["connectionID"] = connectionID
	}

	if len(resourceCollections) > 0 {
		idx = types.ResourceCollectionsFindingsIndex
		terms["resourceCollection"] = resourceCollections
	}

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
		strConformanceStatus := make([]string, 0)
		for _, cr := range conformanceStatuses {
			strConformanceStatus = append(strConformanceStatus, string(cr))
		}
		terms["conformanceStatus"] = strConformanceStatus
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
	}
	root["aggs"] = aggs

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
		logger.Error("FindingsFiltersQuery", zap.Error(err), zap.String("query", string(queryBytes)), zap.String("index", idx))
		return nil, err
	}

	var resp FindingFiltersAggregationResponse
	err = client.Search(context.Background(), idx, string(queryBytes), &resp)
	if err != nil {
		logger.Error("FindingsFiltersQuery", zap.Error(err), zap.String("query", string(queryBytes)), zap.String("index", idx))
		return nil, err
	}
	if len(resourceCollections) == 0 {
		var rcResp FindingFiltersAggregationResponse
		err = client.Search(context.Background(), types.ResourceCollectionsFindingsIndex, string(queryBytes), &rcResp)
		if err != nil {
			logger.Error("FindingsFiltersQuery", zap.Error(err), zap.String("query", string(queryBytes)), zap.String("index", idx))
			return nil, err
		}
		resp.Aggregations.ResourceCollectionFilter = rcResp.Aggregations.ResourceCollectionFilter
	}

	return &resp, nil
}

type Bucket struct {
	Key      string `json:"key"`
	DocCount int    `json:"doc_count"`
}

type FindingsTopFieldResponse struct {
	Aggregations struct {
		FieldFilter struct {
			DocCountErrorUpperBound int      `json:"doc_count_error_upper_bound"`
			SumOtherDocCount        int      `json:"sum_other_doc_count"`
			Buckets                 []Bucket `json:"buckets"`
		} `json:"field_filter"`
		BucketCount struct {
			Value int `json:"value"`
		} `json:"bucket_count"`
	} `json:"aggregations"`
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

func FindingsTopFieldQuery(logger *zap.Logger, client kaytu.Client,
	field string, connectors []source.Type, resourceTypeID []string, connectionIDs []string,
	benchmarkID []string, controlID []string, severity []types.FindingSeverity, conformanceStatuses []types.ConformanceStatus, size int) (*FindingsTopFieldResponse, error) {
	terms := make(map[string]any)
	idx := types.FindingsIndex
	if len(benchmarkID) > 0 {
		terms["benchmarkID"] = benchmarkID
	}

	if len(controlID) > 0 {
		terms["controlID"] = controlID
	}

	if len(conformanceStatuses) > 0 {
		terms["conformanceStatus"] = conformanceStatuses
	}

	if len(severity) > 0 {
		terms["severity"] = severity
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

	logger.Info("FindingsTopFieldQuery", zap.String("query", string(queryBytes)), zap.String("index", idx))
	var resp FindingsTopFieldResponse
	err = client.Search(context.Background(), idx, string(queryBytes), &resp)
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

func ResourceTypesFindingsSummary(logger *zap.Logger, client kaytu.Client,
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
	err = client.Search(context.Background(), types.FindingsIndex, string(queryBytes), &resp)
	return &resp, err
}

func FindingsFieldCountByControl(logger *zap.Logger, client kaytu.Client,
	field string, connectors []source.Type, resourceTypeID []string, connectionIDs []string, resourceCollections []string, benchmarkID []string, controlID []string, severity []types.FindingSeverity) (*FindingsFieldCountByControlResponse, error) {
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

	if len(connectionIDs) > 0 {
		terms["connectionID"] = connectionIDs
	}

	if len(resourceTypeID) > 0 {
		terms["resourceType"] = resourceTypeID
	}

	if len(connectors) > 0 {
		terms["connector"] = connectors
	}

	if len(resourceCollections) > 0 {
		idx = types.ResourceCollectionsFindingsIndex
		terms["resourceCollection"] = resourceCollections
	}

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
	err = client.Search(context.Background(), idx, string(queryBytes), &resp)
	return &resp, err
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

func FetchFindingCountPerKaytuResourceIds(logger *zap.Logger, client kaytu.Client, kaytuResourceIds []string,
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
	err = client.Search(context.Background(), types.FindingsIndex, string(queryBytes), &resp)
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

func FetchFindingsPerControlForResourceId(logger *zap.Logger, client kaytu.Client, kaytuResourceId string) ([]types.Finding, error) {
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
	err = client.Search(context.Background(), types.FindingsIndex, string(queryBytes), &resp)
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
