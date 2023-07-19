package es

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"go.uber.org/zap"

	"github.com/kaytu-io/kaytu-util/pkg/source"

	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
)

type FindingsQueryResponse struct {
	Hits FindingsQueryHits `json:"hits"`
}
type FindingsQueryHits struct {
	Total keibi.SearchTotal  `json:"total"`
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

func GetActiveFindings(client keibi.Client, policyID string, from, size int) (*FindingsQueryResponse, error) {
	res := make(map[string]any)
	var filters []any

	filters = append(filters,
		map[string]any{"term": map[string]any{"stateActive": true}},
		map[string]any{"term": map[string]any{"policyID": policyID}},
	)
	res["size"] = size
	res["from"] = from

	res["sort"] = []map[string]any{
		{
			"_id": "desc",
		},
	}

	res["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}
	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	var resp FindingsQueryResponse
	err = client.SearchWithTrackTotalHits(context.Background(), types.FindingsIndex, string(b), &resp, false)
	return &resp, err
}

func FindingsQuery(client keibi.Client,
	resourceIDs []string,
	provider []source.Type,
	connectionID []string,
	benchmarkID []string,
	policyID []string,
	severity []string,
	sort []map[string]any,
	from, size int) (*FindingsQueryResponse, error) {

	res := make(map[string]any)
	var filters []any

	if len(resourceIDs) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"resourceID": resourceIDs},
		})
	}

	if len(benchmarkID) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"benchmarkID": benchmarkID},
		})
	}

	if len(policyID) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"policyID": policyID},
		})
	}

	if len(severity) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string]any{"severity": severity},
		})
	}

	if len(connectionID) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string]any{"connectionID": connectionID},
		})
	}

	if len(provider) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string]any{"connector": provider},
		})
	}
	res["size"] = size
	res["from"] = from

	if sort != nil && len(sort) > 0 {
		res["sort"] = sort
	}

	res["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}
	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	isStack := false
	if len(connectionID) > 0 {
		if strings.HasPrefix(connectionID[0], "stack-") {
			isStack = true
		}
	}

	var resp FindingsQueryResponse
	if isStack {
		err = client.SearchWithTrackTotalHits(context.Background(), types.StackFindingsIndex, string(b), &resp, true)
	} else {
		err = client.SearchWithTrackTotalHits(context.Background(), types.FindingsIndex, string(b), &resp, true)
	}
	return &resp, err
}

type FindingFiltersAggregationResponse struct {
	Aggregations FindingFiltersAggregations `json:"aggregations"`
}
type FindingFiltersAggregations struct {
	BenchmarkIDFilter  AggregationResult `json:"benchmark_id_filter"`
	PolicyIDFilter     AggregationResult `json:"policy_id_filter"`
	StatusFilter       AggregationResult `json:"status_filter"`
	SeverityFilter     AggregationResult `json:"severity_filter"`
	SourceIDFilter     AggregationResult `json:"source_id_filter"`
	ResourceTypeFilter AggregationResult `json:"resource_type_filter"`
	SourceTypeFilter   AggregationResult `json:"source_type_filter"`
}
type AggregationResult struct {
	DocCountErrorUpperBound int      `json:"doc_count_error_upper_bound"`
	SumOtherDocCount        int      `json:"sum_other_doc_count"`
	Buckets                 []Bucket `json:"buckets"`
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

func FindingsTopFieldQuery(logger *zap.Logger, client keibi.Client,
	field string, connectors []source.Type, resourceTypeID []string, connectionIDs []string,
	benchmarkID []string, policyID []string, severity []types.FindingSeverity, size int) (*FindingsTopFieldResponse, error) {
	terms := make(map[string]any)

	if len(benchmarkID) > 0 {
		terms["benchmarkID"] = benchmarkID
	}

	if len(policyID) > 0 {
		terms["policyID"] = policyID
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

	terms["stateActive"] = []bool{true}

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

	logger.Info("FindingsTopFieldQuery", zap.String("query", string(queryBytes)), zap.String("index", types.FindingsIndex))
	var resp FindingsTopFieldResponse
	err = client.Search(context.Background(), types.FindingsIndex, string(queryBytes), &resp)
	return &resp, err
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
			"term": map[string]string{"benchmarkID": *benchmarkID},
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

	fmt.Println("query=", string(query), "index=", types.FindingsIndex)

	var response LiveBenchmarkAggregatedFindingsQueryResponse
	err = client.Search(context.Background(), types.FindingsIndex, string(query), &response)
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
