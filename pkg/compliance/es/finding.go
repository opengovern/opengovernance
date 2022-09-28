package es

import (
	"context"
	"encoding/json"
	"strconv"

	"gitlab.com/keibiengine/keibi-engine/pkg/source"

	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
)

const (
	FindingsIndex = "findings"
)

type Status string

const (
	StatusAlarm = "alarm"
	StatusInfo  = "info"
	StatusOK    = "ok"
	StatusSkip  = "skip"
	StatusError = "error"
)

type Finding struct {
	ComplianceJobID        uint        `json:"complianceJobID"`
	ScheduleJobID          uint        `json:"scheduleJobID"`
	ResourceID             string      `json:"resourceID"`
	ResourceName           string      `json:"resourceName"`
	ResourceType           string      `json:"resourceType"`
	ServiceName            string      `json:"serviceName"`
	Category               string      `json:"category"`
	ResourceLocation       string      `json:"resourceLocation"`
	Reason                 string      `json:"reason"`
	Status                 Status      `json:"status"`
	DescribedAt            int64       `json:"describedAt"`
	EvaluatedAt            int64       `json:"evaluatedAt"`
	SourceID               uuid.UUID   `json:"sourceID"`
	ConnectionProviderID   string      `json:"connectionProviderID"`
	ConnectionProviderName string      `json:"connectionProviderName"`
	SourceType             source.Type `json:"sourceType"`
	BenchmarkID            string      `json:"benchmarkID"`
	PolicyID               string      `json:"policyID"`
	PolicySeverity         string      `json:"policySeverity"`
}

func (r Finding) KeysAndIndex() ([]string, string) {
	return []string{
		r.ResourceID,
		r.SourceID.String(),
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

func FindingsQuery(client keibi.Client,
	provider []source.Type,
	resourceTypeID []string,
	sourceID []uuid.UUID,
	status []Status,
	benchmarkID []string,
	policyID []string,
	severity []string,
	sort []map[string]interface{},
	from, size int) (*FindingsQueryResponse, error) {

	res := make(map[string]interface{})
	var filters []interface{}

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

	if len(status) > 0 {
		filters = append(filters, map[string]interface{}{
			"terms": map[string]interface{}{"status": status},
		})
	}

	if len(severity) > 0 {
		filters = append(filters, map[string]interface{}{
			"terms": map[string]interface{}{"severity": severity},
		})
	}

	if len(sourceID) > 0 {
		filters = append(filters, map[string]interface{}{
			"terms": map[string]interface{}{"sourceID": sourceID},
		})
	}

	if len(resourceTypeID) > 0 {
		filters = append(filters, map[string]interface{}{
			"terms": map[string]interface{}{"resourceType": resourceTypeID},
		})
	}

	if len(provider) > 0 {
		filters = append(filters, map[string]interface{}{
			"terms": map[string]interface{}{"sourceType": provider},
		})
	}
	res["size"] = size
	res["from"] = from

	res["sort"] = sort

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
	err = client.Search(context.Background(), FindingsIndex, string(b), &res)
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

func FindingsFiltersQuery(client keibi.Client,
	provider []source.Type,
	resourceTypeID []string,
	sourceID []uuid.UUID,
	status []Status,
	benchmarkID []string,
	policyID []string,
	severity []string,
) (*FindingFiltersAggregationResponse, error) {
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
		terms["severity"] = severity
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

	root := map[string]interface{}{}
	root["size"] = 0

	benchmarkIDFilter := map[string]interface{}{
		"terms": map[string]interface{}{"field": "benchmarkID", "size": 1000},
	}
	policyIDFilter := map[string]interface{}{
		"terms": map[string]interface{}{"field": "policyID", "size": 1000},
	}
	statusFilter := map[string]interface{}{
		"terms": map[string]interface{}{"field": "status", "size": 1000},
	}
	severityFilter := map[string]interface{}{
		"terms": map[string]interface{}{"field": "severity", "size": 1000},
	}
	sourceIDFilter := map[string]interface{}{
		"terms": map[string]interface{}{"field": "sourceID", "size": 1000},
	}
	resourceTypeFilter := map[string]interface{}{
		"terms": map[string]interface{}{"field": "resourceType", "size": 1000},
	}
	sourceTypeFilter := map[string]interface{}{
		"terms": map[string]interface{}{"field": "sourceType", "size": 1000},
	}
	aggs := map[string]interface{}{
		"benchmark_id_filter":  benchmarkIDFilter,
		"policy_id_filter":     policyIDFilter,
		"status_filter":        statusFilter,
		"severity_filter":      severityFilter,
		"source_id_filter":     sourceIDFilter,
		"resource_type_filter": resourceTypeFilter,
		"source_type_filter":   sourceTypeFilter,
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

	var resp FindingFiltersAggregationResponse
	err = client.Search(context.Background(), FindingsIndex, string(queryBytes), &resp)
	return &resp, err
}

func FindingsByBenchmarkID(benchmarkID string, status *string, reportID uint, size int, searchAfter []interface{}) (string, error) {
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"parentBenchmarkIDs": {benchmarkID}},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]interface{}{"reportID": {reportID}},
	})
	if status != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"status": {*status}},
		})
	}
	res["size"] = size
	res["sort"] = []map[string]interface{}{
		{
			"_id": "asc",
		},
	}
	if searchAfter != nil {
		res["search_after"] = searchAfter
	}

	res["query"] = map[string]interface{}{
		"bool": map[string]interface{}{
			"filter": filters,
		},
	}
	b, err := json.Marshal(res)
	return string(b), err
}

func FindingsByPolicyID(benchmarkID string, policyID string, reportID uint, size int, searchAfter []interface{}) (string, error) {
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"parentBenchmarkIDs": {benchmarkID}},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]interface{}{"reportID": {reportID}},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"controlID": {policyID}},
	})
	res["size"] = size
	res["sort"] = []map[string]interface{}{
		{
			"_id": "asc",
		},
	}
	if searchAfter != nil {
		res["search_after"] = searchAfter
	}

	res["query"] = map[string]interface{}{
		"bool": map[string]interface{}{
			"filter": filters,
		},
	}
	b, err := json.Marshal(res)
	return string(b), err
}
