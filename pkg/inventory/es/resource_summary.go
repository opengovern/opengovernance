package es

import (
	"encoding/json"
	"strconv"

	"gitlab.com/keibiengine/keibi-engine/pkg/describe/kafka"

	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
)

type ResourceGrowthQueryResponse struct {
	Hits ResourceGrowthQueryHits `json:"hits"`
}
type ResourceGrowthQueryHits struct {
	Total keibi.SearchTotal        `json:"total"`
	Hits  []ResourceGrowthQueryHit `json:"hits"`
}
type ResourceGrowthQueryHit struct {
	ID      string                       `json:"_id"`
	Score   float64                      `json:"_score"`
	Index   string                       `json:"_index"`
	Type    string                       `json:"_type"`
	Version int64                        `json:"_version,omitempty"`
	Source  kafka.SourceResourcesSummary `json:"_source"`
	Sort    []interface{}                `json:"sort"`
}

func FindResourceGrowthTrendQuery(sourceID *uuid.UUID, provider *string,
	createdAtFrom, createdAtTo int64, fetchSize int, searchAfter []interface{}) (string, error) {

	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"report_type": {kafka.ResourceSummaryTypeResourceGrowthTrend}},
	})

	if provider != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_type": {*provider}},
		})
	}

	if sourceID != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_id": {sourceID.String()}},
		})
	}

	filters = append(filters, map[string]interface{}{
		"range": map[string]interface{}{
			"described_at": map[string]string{
				"gte": strconv.FormatInt(createdAtFrom, 10),
				"lte": strconv.FormatInt(createdAtTo, 10),
			},
		},
	})

	res["size"] = fetchSize
	res["sort"] = []map[string]interface{}{
		{
			"described_at": "asc",
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

func FindTopAccountsQuery(provider string, fetchSize int) (string, error) {
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"report_type": {kafka.ResourceSummaryTypeLastSummary}},
	})

	if provider != "" {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_type": {provider}},
		})
	}

	res["size"] = fetchSize
	res["sort"] = []map[string]interface{}{
		{
			"resource_count": "desc",
		},
	}
	res["query"] = map[string]interface{}{
		"bool": map[string]interface{}{
			"filter": filters,
		},
	}
	b, err := json.Marshal(res)
	return string(b), err
}

type TopServicesQueryResponse struct {
	Hits TopServicesQueryHits `json:"hits"`
}
type TopServicesQueryHits struct {
	Total keibi.SearchTotal     `json:"total"`
	Hits  []TopServicesQueryHit `json:"hits"`
}
type TopServicesQueryHit struct {
	ID      string                      `json:"_id"`
	Score   float64                     `json:"_score"`
	Index   string                      `json:"_index"`
	Type    string                      `json:"_type"`
	Version int64                       `json:"_version,omitempty"`
	Source  kafka.SourceServicesSummary `json:"_source"`
	Sort    []interface{}               `json:"sort"`
}

func FindTopServicesQuery(provider string, sourceID *string, fetchSize int) (string, error) {
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"report_type": {kafka.ResourceSummaryTypeLastServiceSummary}},
	})

	if provider != "" {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_type": {provider}},
		})
	}

	if sourceID != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_id": {*sourceID}},
		})
	}

	res["size"] = fetchSize
	res["sort"] = []map[string]interface{}{
		{
			"resource_count": "desc",
		},
	}
	res["query"] = map[string]interface{}{
		"bool": map[string]interface{}{
			"filter": filters,
		},
	}
	b, err := json.Marshal(res)
	return string(b), err
}

type CategoriesQueryResponse struct {
	Hits CategoriesQueryHits `json:"hits"`
}
type CategoriesQueryHits struct {
	Total keibi.SearchTotal    `json:"total"`
	Hits  []CategoriesQueryHit `json:"hits"`
}
type CategoriesQueryHit struct {
	ID      string                      `json:"_id"`
	Score   float64                     `json:"_score"`
	Index   string                      `json:"_index"`
	Type    string                      `json:"_type"`
	Version int64                       `json:"_version,omitempty"`
	Source  kafka.SourceCategorySummary `json:"_source"`
	Sort    []interface{}               `json:"sort"`
}

func GetCategoriesQuery(provider string, fetchSize int) (string, error) {
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"report_type": {kafka.ResourceSummaryTypeLastCategorySummary}},
	})

	if provider != "" {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_type": {provider}},
		})
	}

	res["size"] = fetchSize
	res["sort"] = []map[string]interface{}{
		{
			"resource_count": "desc",
		},
	}
	res["query"] = map[string]interface{}{
		"bool": map[string]interface{}{
			"filter": filters,
		},
	}
	b, err := json.Marshal(res)
	return string(b), err
}

func ListAccountResourceCountQuery(provider string, fetchSize int, searchAfter []interface{}) (string, error) {
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"report_type": {kafka.ResourceSummaryTypeLastSummary}},
	})

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"source_type": {provider}},
	})

	res["size"] = fetchSize
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

type LocationDistributionQueryResponse struct {
	Hits LocationDistributionQueryHits `json:"hits"`
}
type LocationDistributionQueryHits struct {
	Total keibi.SearchTotal              `json:"total"`
	Hits  []LocationDistributionQueryHit `json:"hits"`
}
type LocationDistributionQueryHit struct {
	ID      string                             `json:"_id"`
	Score   float64                            `json:"_score"`
	Index   string                             `json:"_index"`
	Type    string                             `json:"_type"`
	Version int64                              `json:"_version,omitempty"`
	Source  kafka.LocationDistributionResource `json:"_source"`
	Sort    []interface{}                      `json:"sort"`
}

type ServiceDistributionQueryResponse struct {
	Hits ServiceDistributionQueryHits `json:"hits"`
}
type ServiceDistributionQueryHits struct {
	Total keibi.SearchTotal             `json:"total"`
	Hits  []ServiceDistributionQueryHit `json:"hits"`
}
type ServiceDistributionQueryHit struct {
	ID      string                                  `json:"_id"`
	Score   float64                                 `json:"_score"`
	Index   string                                  `json:"_index"`
	Type    string                                  `json:"_type"`
	Version int64                                   `json:"_version,omitempty"`
	Source  kafka.SourceServiceDistributionResource `json:"_source"`
	Sort    []interface{}                           `json:"sort"`
}

func FindLocationDistributionQuery(sourceID *uuid.UUID, provider *string,
	fetchSize int, searchAfter []interface{}) (string, error) {

	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"report_type": {kafka.ResourceSummaryTypeLocationDistribution}},
	})

	if provider != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_type": {*provider}},
		})
	}

	if sourceID != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_id": {sourceID.String()}},
		})
	}

	res["size"] = fetchSize
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

func FindSourceServiceDistributionQuery(sourceID uuid.UUID, fetchSize int, searchAfter []interface{}) (string, error) {
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"report_type": {kafka.ResourceSummaryTypeServiceDistributionSummary}},
	})

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"source_id": {sourceID.String()}},
	})

	res["size"] = fetchSize
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

type ComplianceTrendQueryResponse struct {
	Hits ComplianceTrendQueryHits `json:"hits"`
}
type ComplianceTrendQueryHits struct {
	Total keibi.SearchTotal         `json:"total"`
	Hits  []ComplianceTrendQueryHit `json:"hits"`
}
type ComplianceTrendQueryHit struct {
	ID      string                                `json:"_id"`
	Score   float64                               `json:"_score"`
	Index   string                                `json:"_index"`
	Type    string                                `json:"_type"`
	Version int64                                 `json:"_version,omitempty"`
	Source  kafka.ResourceCompliancyTrendResource `json:"_source"`
	Sort    []interface{}                         `json:"sort"`
}

func FindCompliancyTrendQuery(sourceID *uuid.UUID, provider *string,
	describedAtFrom, describedAtTo int64, fetchSize int, searchAfter []interface{}) (string, error) {

	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"report_type": {kafka.ResourceSummaryTypeCompliancyTrend}},
	})

	if provider != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_type": {*provider}},
		})
	}

	if sourceID != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_id": {sourceID.String()}},
		})
	}

	filters = append(filters, map[string]interface{}{
		"range": map[string]interface{}{
			"described_at": map[string]string{
				"gte": strconv.FormatInt(describedAtFrom, 10),
				"lte": strconv.FormatInt(describedAtTo, 10),
			},
		},
	})

	res["size"] = fetchSize
	res["sort"] = []map[string]interface{}{
		{
			"described_at": "asc",
		},
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
