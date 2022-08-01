package es

import (
	"encoding/json"
	"fmt"
	"strconv"

	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/api"

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
		{
			"_id": "desc",
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

type ResourceTypeQueryResponse struct {
	Hits ResourceTypeQueryHits `json:"hits"`
}
type ResourceTypeQueryHits struct {
	Total keibi.SearchTotal      `json:"total"`
	Hits  []ResourceTypeQueryHit `json:"hits"`
}
type ResourceTypeQueryHit struct {
	ID      string                       `json:"_id"`
	Score   float64                      `json:"_score"`
	Index   string                       `json:"_index"`
	Type    string                       `json:"_type"`
	Version int64                        `json:"_version,omitempty"`
	Source  kafka.SourceResourcesSummary `json:"_source"`
	Sort    []interface{}                `json:"sort"`
}

func GetResourceTypeQuery(provider string, sourceID *string, resourceTypes []string,
	fetchSize int, searchAfter []interface{}) (string, error) {

	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"report_type": {kafka.ResourceSummaryTypeLastSummary}},
	})

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"resource_type": resourceTypes},
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
	if searchAfter != nil {
		res["search_after"] = searchAfter
	}
	res["sort"] = []map[string]interface{}{
		{
			"resource_count": "desc",
		},
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
	return string(b), err
}

func FindTopAccountsQuery(provider string, fetchSize int, searchAfter []interface{}) (string, error) {
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

	if searchAfter != nil {
		res["search_after"] = searchAfter
	}

	res["size"] = fetchSize
	res["sort"] = []map[string]interface{}{
		{
			"resource_count": "desc",
		},
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
	return string(b), err
}

type FetchServicesQueryResponse struct {
	Hits FetchServicesQueryHits `json:"hits"`
}
type FetchServicesQueryHits struct {
	Total keibi.SearchTotal       `json:"total"`
	Hits  []FetchServicesQueryHit `json:"hits"`
}
type FetchServicesQueryHit struct {
	ID      string                      `json:"_id"`
	Score   float64                     `json:"_score"`
	Index   string                      `json:"_index"`
	Type    string                      `json:"_type"`
	Version int64                       `json:"_version,omitempty"`
	Source  kafka.SourceServicesSummary `json:"_source"`
	Sort    []interface{}               `json:"sort"`
}

func FetchServicesQuery(provider string, sourceID *string, fetchSize int, searchAfter []interface{}) (string, error) {
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
	if searchAfter != nil {
		res["search_after"] = searchAfter
	}

	res["sort"] = []map[string]interface{}{
		{
			"resource_count": "desc",
		},
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

func GetCategoriesQuery(provider string, sourceID *string, fetchSize int, searchAfter []interface{}) (string, error) {
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

	if sourceID != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_id": {*sourceID}},
		})
	}

	res["size"] = fetchSize
	if searchAfter != nil {
		res["search_after"] = searchAfter
	}
	res["sort"] = []map[string]interface{}{
		{
			"resource_count": "desc",
		},
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

	if provider != nil && *provider != "all" {
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

func FindAWSCostQuery(sourceID *uuid.UUID, fetchSize int, searchAfter []interface{}) (string, error) {
	res := make(map[string]interface{})

	res["size"] = fetchSize
	res["sort"] = []map[string]interface{}{
		{
			"_id": "asc",
		},
	}
	if searchAfter != nil {
		res["search_after"] = searchAfter
	}

	if sourceID != nil {
		var filters []interface{}
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_id": {sourceID.String()}},
		})
		res["query"] = map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": filters,
			},
		}
	}
	b, err := json.Marshal(res)
	return string(b), err
}

type LookupResourceAggregationResponse struct {
	Aggregations LookupResourceAggregations `json:"aggregations"`
}
type LookupResourceAggregations struct {
	ResourceTypeFilter AggregationResult `json:"resource_type_filter"`
	SourceTypeFilter   AggregationResult `json:"source_type_filter"`
	CategoryFilter     AggregationResult `json:"category_filter"`
	ServiceFilter      AggregationResult `json:"service_filter"`
	LocationFilter     AggregationResult `json:"location_filter"`
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

func BuildFilterQuery(
	query string,
	filters api.ResourceFilters,
	commonFilter *bool,
) (string, error) {
	terms := make(map[string][]string)
	if !api.FilterIsEmpty(filters.Location) {
		terms["location"] = filters.Location
	}

	if !api.FilterIsEmpty(filters.ResourceType) {
		terms["resource_type"] = filters.ResourceType
	}

	if !api.FilterIsEmpty(filters.Category) {
		terms["category"] = filters.Category
	}

	if !api.FilterIsEmpty(filters.Service) {
		terms["service_name"] = filters.Service
	}

	if !api.FilterIsEmpty(filters.Provider) {
		terms["source_type"] = filters.Provider
	}

	if commonFilter != nil {
		terms["is_common"] = []string{fmt.Sprintf("%v", *commonFilter)}
	}

	notTerms := make(map[string][]string)
	ignoreResourceTypes := []string{
		"Microsoft.Resources/subscriptions/locations",
		"Microsoft.Authorization/roleDefinitions",
		"microsoft.security/autoProvisioningSettings",
		"microsoft.security/settings",
		"Microsoft.Authorization/elevateAccessRoleAssignment",
		"Microsoft.AppConfiguration/configurationStores",
		"Microsoft.KeyVault/vaults/keys",
		"microsoft.security/pricings",
		"Microsoft.Security/autoProvisioningSettings",
		"Microsoft.Security/securityContacts",
		"Microsoft.Security/locations/jitNetworkAccessPolicies",
		"AWS::EC2::Region",
		"AWS::EC2::RegionalSettings",
	}
	notTerms["resource_type"] = ignoreResourceTypes

	root := map[string]interface{}{}
	root["size"] = 0

	sourceTypeFilter := map[string]interface{}{
		"terms": map[string]interface{}{"field": "source_type", "size": 1000},
	}
	categoryFilter := map[string]interface{}{
		"terms": map[string]interface{}{"field": "category", "size": 1000},
	}
	serviceFilter := map[string]interface{}{
		"terms": map[string]interface{}{"field": "service_name", "size": 1000},
	}
	resourceTypeFilter := map[string]interface{}{
		"terms": map[string]interface{}{"field": "resource_type", "size": 1000},
	}
	locationFilter := map[string]interface{}{
		"terms": map[string]interface{}{"field": "location", "size": 1000},
	}
	aggs := map[string]interface{}{
		"source_type_filter":   sourceTypeFilter,
		"category_filter":      categoryFilter,
		"service_filter":       serviceFilter,
		"resource_type_filter": resourceTypeFilter,
		"location_filter":      locationFilter,
	}
	root["aggs"] = aggs

	boolQuery := make(map[string]interface{})
	if terms != nil && len(terms) > 0 {
		var filters []map[string]interface{}
		for k, vs := range terms {
			filters = append(filters, map[string]interface{}{
				"terms": map[string][]string{
					k: vs,
				},
			})
		}

		boolQuery["filter"] = filters
	}
	if len(query) > 0 {
		boolQuery["must"] = map[string]interface{}{
			"multi_match": map[string]interface{}{
				"fields": []string{"resource_id", "name", "source_type", "resource_type", "resource_group",
					"location", "source_id"},
				"query":     query,
				"fuzziness": "AUTO",
			},
		}
	}
	if len(notTerms) > 0 {
		var filters []map[string]interface{}
		for k, vs := range notTerms {
			filters = append(filters, map[string]interface{}{
				"terms": map[string][]string{
					k: vs,
				},
			})
		}

		boolQuery["must_not"] = filters
	}
	if len(boolQuery) > 0 {
		root["query"] = map[string]interface{}{
			"bool": boolQuery,
		}
	}

	queryBytes, err := json.Marshal(root)
	if err != nil {
		return "", err
	}
	return string(queryBytes), nil
}
