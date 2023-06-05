package es

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"gitlab.com/keibiengine/keibi-engine/pkg/types"

	"github.com/kaytu-io/kaytu-util/pkg/source"

	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
)

const (
	FindingsIndex = "findings"
)

type Finding struct {
	ID               string                 `json:"ID"`
	BenchmarkID      string                 `json:"benchmarkID"`
	PolicyID         string                 `json:"policyID"`
	ConnectionID     string                 `json:"connectionID"`
	DescribedAt      int64                  `json:"describedAt"`
	EvaluatedAt      int64                  `json:"evaluatedAt"`
	StateActive      bool                   `json:"stateActive"`
	Result           types.ComplianceResult `json:"result"`
	Severity         types.Severity         `json:"severity"`
	Evaluator        string                 `json:"evaluator"`
	Connector        source.Type            `json:"connector"`
	ResourceID       string                 `json:"resourceID"`
	ResourceName     string                 `json:"resourceName"`
	ResourceLocation string                 `json:"resourceLocation"`
	ResourceType     string                 `json:"resourceType"`
	Reason           string                 `json:"reason"`
	ComplianceJobID  uint                   `json:"complianceJobID"`
	ScheduleJobID    uint                   `json:"scheduleJobID"`
}

func (r Finding) KeysAndIndex() ([]string, string) {
	return []string{
		r.ResourceID,
		r.ConnectionID,
		r.PolicyID,
		strconv.FormatInt(r.DescribedAt, 10),
	}, FindingsIndex
}

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
	Source  Finding       `json:"_source"`
	Sort    []interface{} `json:"sort"`
}

func GetActiveFindings(client keibi.Client, from, size int) (*FindingsQueryResponse, error) {
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"term": map[string]interface{}{"stateActive": true},
	})
	res["size"] = size
	res["from"] = from

	res["sort"] = []map[string]interface{}{
		{
			"_id": "desc",
		},
	}

	res["query"] = map[string]interface{}{
		"bool": map[string]interface{}{
			"filter": filters,
		},
	}
	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	var resp FindingsQueryResponse
	err = client.SearchWithTrackTotalHits(context.Background(), FindingsIndex, string(b), &resp, false)
	return &resp, err
}

func FindingsQuery(client keibi.Client,
	id []string,
	provider []source.Type,
	resourceID []string,
	connectionID []string,
	benchmarkID []string,
	policyID []string,
	severity []string,
	sort []map[string]interface{},
	from, size int) (*FindingsQueryResponse, error) {

	res := make(map[string]interface{})
	var filters []interface{}

	if len(id) > 0 {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"id": id},
		})
	}

	if len(benchmarkID) > 0 {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"benchmarkID": benchmarkID},
		})
	}

	if len(policyID) > 0 {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"policyID": policyID},
		})
	}

	if len(severity) > 0 {
		filters = append(filters, map[string]interface{}{
			"terms": map[string]interface{}{"policySeverity": severity},
		})
	}

	if len(connectionID) > 0 {
		filters = append(filters, map[string]interface{}{
			"terms": map[string]interface{}{"connectionID": connectionID},
		})
	}

	if len(resourceID) > 0 {
		filters = append(filters, map[string]interface{}{
			"terms": map[string]interface{}{"resourceID": resourceID},
		})
	}

	if len(provider) > 0 {
		filters = append(filters, map[string]interface{}{
			"terms": map[string]interface{}{"sourceType": provider},
		})
	}
	res["size"] = size
	res["from"] = from

	if sort != nil && len(sort) > 0 {
		res["sort"] = sort
	}

	res["query"] = map[string]interface{}{
		"bool": map[string]interface{}{
			"filter": filters,
		},
	}
	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	var resp FindingsQueryResponse
	err = client.SearchWithTrackTotalHits(context.Background(), FindingsIndex, string(b), &resp, true)
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
	Aggregations FindingsTopFieldAggregations `json:"aggregations"`
}
type FindingsTopFieldAggregations struct {
	FieldFilter AggregationResult `json:"field_filter"`
}

func FindingsTopFieldQuery(client keibi.Client,
	field string,
	provider []source.Type,
	resourceTypeID []string,
	sourceID []string,
	status []types.ComplianceResult,
	benchmarkID []string,
	policyID []string,
	severity []string,
	size int,
) (*FindingsTopFieldResponse, error) {
	terms := make(map[string]interface{})

	if len(benchmarkID) > 0 {
		terms["benchmarkID"] = benchmarkID
	}

	if len(policyID) > 0 {
		terms["policyID"] = policyID
	}

	if len(status) > 0 {
		terms["status"] = status
	}

	if len(severity) > 0 {
		terms["policySeverity"] = severity
	}

	if len(sourceID) > 0 {
		terms["sourceID"] = sourceID
	}

	if len(resourceTypeID) > 0 {
		terms["resourceType"] = resourceTypeID
	}

	if len(provider) > 0 {
		terms["sourceType"] = provider
	}

	terms["stateActive"] = []bool{true}

	root := map[string]interface{}{}
	root["size"] = 0

	fieldFilter := map[string]interface{}{
		"terms": map[string]interface{}{"field": field, "size": size},
	}
	aggs := map[string]interface{}{
		"field_filter": fieldFilter,
	}
	root["aggs"] = aggs

	boolQuery := make(map[string]interface{})
	if terms != nil && len(terms) > 0 {
		var filters []map[string]interface{}
		for k, vs := range terms {
			filters = append(filters, map[string]interface{}{
				"terms": map[string]interface{}{
					k: vs,
				},
			})
		}

		boolQuery["filter"] = filters
	}
	if len(boolQuery) > 0 {
		root["query"] = map[string]interface{}{
			"bool": boolQuery,
		}
	}

	queryBytes, err := json.Marshal(root)
	if err != nil {
		return nil, err
	}

	fmt.Println("======", string(queryBytes))
	var resp FindingsTopFieldResponse
	err = client.Search(context.Background(), FindingsIndex, string(queryBytes), &resp)
	return &resp, err
}
