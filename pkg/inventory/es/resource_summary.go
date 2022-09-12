package es

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"gitlab.com/keibiengine/keibi-engine/pkg/source"

	kafka2 "gitlab.com/keibiengine/keibi-engine/pkg/summarizer/kafka"

	"gitlab.com/keibiengine/keibi-engine/pkg/describe"

	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/api"

	"gitlab.com/keibiengine/keibi-engine/pkg/describe/kafka"

	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
)

const EsFetchPageSize = 10000

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

func FindResourceGrowthTrend(client keibi.Client, sourceID *uuid.UUID, provider source.Type,
	createdAtFrom, createdAtTo int64) ([]kafka.SourceResourcesSummary, error) {

	var hits []kafka.SourceResourcesSummary
	var searchAfter []interface{}
	for {
		res := make(map[string]interface{})

		var filters []interface{}
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"report_type": {kafka.ResourceSummaryTypeResourceGrowthTrend}},
		})

		if !provider.IsNull() {
			filters = append(filters, map[string]interface{}{
				"terms": map[string][]string{"source_type": {provider.String()}},
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

		res["size"] = EsFetchPageSize
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
		if err != nil {
			return nil, err
		}

		query := string(b)

		var response ResourceGrowthQueryResponse
		err = client.Search(context.Background(), describe.SourceResourcesSummary, query, &response)
		if err != nil {
			return nil, err
		}

		if len(response.Hits.Hits) == 0 {
			break
		}

		for _, hit := range response.Hits.Hits {
			hits = append(hits, hit.Source)
			searchAfter = hit.Sort
		}
	}
	return hits, nil
}

func FetchResourceLastSummary(client keibi.Client, provider source.Type, sourceID *string, resourceType *string) ([]kafka.SourceResourcesSummary, error) {
	var hits []kafka.SourceResourcesSummary
	var searchAfter []interface{}
	for {
		res := make(map[string]interface{})
		var filters []interface{}

		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"report_type": {kafka.ResourceSummaryTypeLastSummary}},
		})

		if !provider.IsNull() {
			filters = append(filters, map[string]interface{}{
				"terms": map[string][]string{"source_type": {provider.String()}},
			})
		}

		if sourceID != nil {
			filters = append(filters, map[string]interface{}{
				"terms": map[string][]string{"source_id": {*sourceID}},
			})
		}

		if resourceType != nil {
			filters = append(filters, map[string]interface{}{
				"terms": map[string][]string{"resource_type": {*resourceType}},
			})
		}

		if searchAfter != nil {
			res["search_after"] = searchAfter
		}

		res["size"] = EsFetchPageSize
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

		query := string(b)

		var response ResourceGrowthQueryResponse
		err = client.Search(context.Background(), describe.SourceResourcesSummary, query, &response)
		if err != nil {
			return nil, err
		}

		if len(response.Hits.Hits) == 0 {
			break
		}

		for _, hit := range response.Hits.Hits {
			searchAfter = hit.Sort
			hits = append(hits, hit.Source)
		}
	}

	//TODO-Saleh also put it in the cache
	return hits, nil
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

func FetchServicesQuery(client keibi.Client, provider string, sourceID *string) ([]kafka.SourceServicesSummary, error) {
	var hits []kafka.SourceServicesSummary
	var searchAfter []interface{}
	for {
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

		res["size"] = EsFetchPageSize
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
		if err != nil {
			return nil, err
		}

		query := string(b)

		var response FetchServicesQueryResponse
		err = client.Search(context.Background(), describe.SourceResourcesSummary, query, &response)
		if err != nil {
			return nil, err
		}

		if len(response.Hits.Hits) == 0 {
			break
		}

		for _, hit := range response.Hits.Hits {
			hits = append(hits, hit.Source)
			searchAfter = hit.Sort
		}
	}
	return hits, nil
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

func GetCategoriesQuery(client keibi.Client, provider string, sourceID *string) ([]kafka.SourceCategorySummary, error) {
	var hits []kafka.SourceCategorySummary
	var searchAfter []interface{}
	for {
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

		res["size"] = EsFetchPageSize
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
		if err != nil {
			return nil, err
		}
		query := string(b)

		var response CategoriesQueryResponse
		err = client.Search(context.Background(), describe.SourceResourcesSummary, query, &response)
		if err != nil {
			return nil, err
		}

		if len(response.Hits.Hits) == 0 {
			break
		}

		for _, hit := range response.Hits.Hits {
			hits = append(hits, hit.Source)
			searchAfter = hit.Sort
		}
	}
	return hits, nil
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

func FindLocationDistributionQuery(client keibi.Client, provider source.Type, sourceID *uuid.UUID) ([]kafka.LocationDistributionResource, error) {
	var hits []kafka.LocationDistributionResource

	var searchAfter []interface{}
	for {
		res := make(map[string]interface{})
		var filters []interface{}

		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"report_type": {kafka.ResourceSummaryTypeLocationDistribution}},
		})

		if !provider.IsNull() {
			filters = append(filters, map[string]interface{}{
				"terms": map[string][]string{"source_type": {provider.String()}},
			})
		}

		if sourceID != nil {
			filters = append(filters, map[string]interface{}{
				"terms": map[string][]string{"source_id": {sourceID.String()}},
			})
		}

		res["size"] = EsFetchPageSize
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
		if err != nil {
			return nil, err
		}
		query := string(b)

		var response LocationDistributionQueryResponse
		err = client.Search(context.Background(), describe.SourceResourcesSummary, query, &response)
		if err != nil {
			return nil, err
		}

		if len(response.Hits.Hits) == 0 {
			break
		}

		for _, hit := range response.Hits.Hits {
			hits = append(hits, hit.Source)
			searchAfter = hit.Sort
		}
	}
	return hits, nil
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

func FindCompliancyTrendQuery(sourceID *uuid.UUID, provider source.Type,
	describedAtFrom, describedAtTo int64, fetchSize int, searchAfter []interface{}) (string, error) {

	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"report_type": {kafka.ResourceSummaryTypeCompliancyTrend}},
	})

	if !provider.IsNull() {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_type": {provider.String()}},
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

type ConnectionResourcesSummaryQueryResponse struct {
	Hits ConnectionResourcesSummaryQueryHits `json:"hits"`
}
type ConnectionResourcesSummaryQueryHits struct {
	Total keibi.SearchTotal                    `json:"total"`
	Hits  []ConnectionResourcesSummaryQueryHit `json:"hits"`
}
type ConnectionResourcesSummaryQueryHit struct {
	ID      string                            `json:"_id"`
	Score   float64                           `json:"_score"`
	Index   string                            `json:"_index"`
	Type    string                            `json:"_type"`
	Version int64                             `json:"_version,omitempty"`
	Source  kafka2.ConnectionResourcesSummary `json:"_source"`
	Sort    []interface{}                     `json:"sort"`
}

func FetchConnectionResourcesSummaryPage(client keibi.Client, provider source.Type, sourceID *string, sort []map[string]interface{}, size int) ([]kafka2.ConnectionResourcesSummary, error) {
	var hits []kafka2.ConnectionResourcesSummary
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"report_type": {kafka.ResourceSummaryTypeLastSummary}},
	})

	if !provider.IsNull() {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_type": {provider.String()}},
		})
	}

	if sourceID != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_id": {*sourceID}},
		})
	}

	sort = append(sort,
		map[string]interface{}{
			"_id": "desc",
		},
	)
	res["size"] = size
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

	query := string(b)

	var response ConnectionResourcesSummaryQueryResponse
	err = client.Search(context.Background(), kafka2.ConnectionSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	for _, hit := range response.Hits.Hits {
		hits = append(hits, hit.Source)
	}
	return hits, nil
}

type ConnectionServicesSummaryQueryResponse struct {
	Hits ConnectionServicesSummaryQueryHits `json:"hits"`
}
type ConnectionServicesSummaryQueryHits struct {
	Total keibi.SearchTotal                   `json:"total"`
	Hits  []ConnectionServicesSummaryQueryHit `json:"hits"`
}
type ConnectionServicesSummaryQueryHit struct {
	ID      string                           `json:"_id"`
	Score   float64                          `json:"_score"`
	Index   string                           `json:"_index"`
	Type    string                           `json:"_type"`
	Version int64                            `json:"_version,omitempty"`
	Source  kafka2.ConnectionServicesSummary `json:"_source"`
	Sort    []interface{}                    `json:"sort"`
}

func FetchConnectionServicesSummaryPage(client keibi.Client, provider source.Type, sort []map[string]interface{}, size int) ([]kafka2.ConnectionServicesSummary, error) {
	var hits []kafka2.ConnectionServicesSummary
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"report_type": {kafka.ResourceSummaryTypeServiceHistorySummary}},
	})

	if !provider.IsNull() {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_type": {provider.String()}},
		})
	}

	sort = append(sort,
		map[string]interface{}{
			"_id": "desc",
		},
	)
	res["size"] = size
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

	query := string(b)

	var response ConnectionServicesSummaryQueryResponse
	err = client.Search(context.Background(), kafka2.ConnectionSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	for _, hit := range response.Hits.Hits {
		hits = append(hits, hit.Source)
	}
	return hits, nil
}

type ConnectionCategoriesSummaryQueryResponse struct {
	Hits ConnectionCategoriesSummaryQueryHits `json:"hits"`
}
type ConnectionCategoriesSummaryQueryHits struct {
	Total keibi.SearchTotal                     `json:"total"`
	Hits  []ConnectionCategoriesSummaryQueryHit `json:"hits"`
}
type ConnectionCategoriesSummaryQueryHit struct {
	ID      string                             `json:"_id"`
	Score   float64                            `json:"_score"`
	Index   string                             `json:"_index"`
	Type    string                             `json:"_type"`
	Version int64                              `json:"_version,omitempty"`
	Source  kafka2.ConnectionCategoriesSummary `json:"_source"`
	Sort    []interface{}                      `json:"sort"`
}

func FetchConnectionCategoriesSummaryPage(client keibi.Client, provider source.Type, sort []map[string]interface{}, size int) ([]kafka2.ConnectionCategoriesSummary, error) {
	var hits []kafka2.ConnectionCategoriesSummary
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"report_type": {kafka.ResourceSummaryTypeCategoryHistorySummary}},
	})

	if !provider.IsNull() {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_type": {provider.String()}},
		})
	}

	sort = append(sort,
		map[string]interface{}{
			"_id": "desc",
		},
	)
	res["size"] = size
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

	query := string(b)

	var response ConnectionCategoriesSummaryQueryResponse
	err = client.Search(context.Background(), kafka2.ConnectionSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	for _, hit := range response.Hits.Hits {
		hits = append(hits, hit.Source)
	}
	return hits, nil
}

type ConnectionLocationsSummaryQueryResponse struct {
	Hits ConnectionLocationsSummaryQueryHits `json:"hits"`
}
type ConnectionLocationsSummaryQueryHits struct {
	Total keibi.SearchTotal                    `json:"total"`
	Hits  []ConnectionLocationsSummaryQueryHit `json:"hits"`
}
type ConnectionLocationsSummaryQueryHit struct {
	ID      string                           `json:"_id"`
	Score   float64                          `json:"_score"`
	Index   string                           `json:"_index"`
	Type    string                           `json:"_type"`
	Version int64                            `json:"_version,omitempty"`
	Source  kafka2.ConnectionLocationSummary `json:"_source"`
	Sort    []interface{}                    `json:"sort"`
}

func FetchConnectionLocationsSummaryPage(client keibi.Client, provider source.Type, sourceID *string, sort []map[string]interface{}, size int) ([]kafka2.ConnectionLocationSummary, error) {
	var hits []kafka2.ConnectionLocationSummary
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"report_type": {kafka.ResourceSummaryTypeLocationDistribution}},
	})

	if !provider.IsNull() {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_type": {provider.String()}},
		})
	}

	if sourceID != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_id": {*sourceID}},
		})
	}

	sort = append(sort,
		map[string]interface{}{
			"_id": "desc",
		},
	)
	res["size"] = size
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

	query := string(b)

	var response ConnectionLocationsSummaryQueryResponse
	err = client.Search(context.Background(), kafka2.ConnectionSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	for _, hit := range response.Hits.Hits {
		hits = append(hits, hit.Source)
	}
	return hits, nil
}
