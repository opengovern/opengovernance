package es

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"gitlab.com/keibiengine/keibi-engine/pkg/insight/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"

	summarizer "gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"

	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/api"

	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
)

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
	ConnectionFilter   AggregationResult `json:"connection_id_filter"`
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

	if !api.FilterIsEmpty(filters.Connections) {
		terms["source_id"] = filters.Connections
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
	connectionIDFilter := map[string]interface{}{
		"terms": map[string]interface{}{"field": "source_id", "size": 1000},
	}
	aggs := map[string]interface{}{
		"source_type_filter":   sourceTypeFilter,
		"category_filter":      categoryFilter,
		"service_filter":       serviceFilter,
		"resource_type_filter": resourceTypeFilter,
		"location_filter":      locationFilter,
		"connection_id_filter": connectionIDFilter,
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
	ID      string                                `json:"_id"`
	Score   float64                               `json:"_score"`
	Index   string                                `json:"_index"`
	Type    string                                `json:"_type"`
	Version int64                                 `json:"_version,omitempty"`
	Source  summarizer.ConnectionResourcesSummary `json:"_source"`
	Sort    []interface{}                         `json:"sort"`
}

func FetchConnectionResourcesSummaryPage(client keibi.Client, provider source.Type, sourceID *string, sort []map[string]interface{}, size int) ([]summarizer.ConnectionResourcesSummary, error) {
	var hits []summarizer.ConnectionResourcesSummary
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"report_type": {string(summarizer.ResourceSummary)}},
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
	err = client.Search(context.Background(), summarizer.ConnectionSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	for _, hit := range response.Hits.Hits {
		hits = append(hits, hit.Source)
	}
	return hits, nil
}

type ProviderServicesSummaryQueryResponse struct {
	Hits ProviderServicesSummaryQueryHits `json:"hits"`
}
type ProviderServicesSummaryQueryHits struct {
	Total keibi.SearchTotal                 `json:"total"`
	Hits  []ProviderServicesSummaryQueryHit `json:"hits"`
}
type ProviderServicesSummaryQueryHit struct {
	ID      string                            `json:"_id"`
	Score   float64                           `json:"_score"`
	Index   string                            `json:"_index"`
	Type    string                            `json:"_type"`
	Version int64                             `json:"_version,omitempty"`
	Source  summarizer.ProviderServiceSummary `json:"_source"`
	Sort    []interface{}                     `json:"sort"`
}

func FetchProviderServicesSummaryPage(client keibi.Client, provider source.Type, sort []map[string]interface{}, size int) ([]summarizer.ProviderServiceSummary, error) {
	var hits []summarizer.ProviderServiceSummary
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"report_type": {string(summarizer.ServiceProviderSummary)}},
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

	var response ProviderServicesSummaryQueryResponse
	err = client.Search(context.Background(), summarizer.ProviderSummaryIndex, query, &response)
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
	ID      string                              `json:"_id"`
	Score   float64                             `json:"_score"`
	Index   string                              `json:"_index"`
	Type    string                              `json:"_type"`
	Version int64                               `json:"_version,omitempty"`
	Source  summarizer.ConnectionServiceSummary `json:"_source"`
	Sort    []interface{}                       `json:"sort"`
}

func FetchConnectionServicesSummaryPage(client keibi.Client, sourceId *string, sort []map[string]interface{}, size int) ([]summarizer.ConnectionServiceSummary, error) {
	var hits []summarizer.ConnectionServiceSummary
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"report_type": {string(summarizer.ServiceSummary)}},
	})

	if sourceId != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_id": {*sourceId}},
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
	err = client.Search(context.Background(), summarizer.ConnectionSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	for _, hit := range response.Hits.Hits {
		hits = append(hits, hit.Source)
	}
	return hits, nil
}

type ProviderCategoriesSummaryQueryResponse struct {
	Hits ProviderCategoriesSummaryQueryHits `json:"hits"`
}
type ProviderCategoriesSummaryQueryHits struct {
	Total keibi.SearchTotal                   `json:"total"`
	Hits  []ProviderCategoriesSummaryQueryHit `json:"hits"`
}
type ProviderCategoriesSummaryQueryHit struct {
	ID      string                             `json:"_id"`
	Score   float64                            `json:"_score"`
	Index   string                             `json:"_index"`
	Type    string                             `json:"_type"`
	Version int64                              `json:"_version,omitempty"`
	Source  summarizer.ProviderCategorySummary `json:"_source"`
	Sort    []interface{}                      `json:"sort"`
}

func FetchProviderCategoriesSummaryPage(client keibi.Client, provider source.Type, sort []map[string]interface{}, size int) ([]summarizer.ProviderCategorySummary, error) {
	var hits []summarizer.ProviderCategorySummary
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"report_type": {string(summarizer.CategoryProviderSummary)}},
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

	var response ProviderCategoriesSummaryQueryResponse
	err = client.Search(context.Background(), summarizer.ProviderSummaryIndex, query, &response)
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
	ID      string                               `json:"_id"`
	Score   float64                              `json:"_score"`
	Index   string                               `json:"_index"`
	Type    string                               `json:"_type"`
	Version int64                                `json:"_version,omitempty"`
	Source  summarizer.ConnectionCategorySummary `json:"_source"`
	Sort    []interface{}                        `json:"sort"`
}

func FetchConnectionCategoriesSummaryPage(client keibi.Client, sourceId *string, sort []map[string]interface{}, size int) ([]summarizer.ConnectionCategorySummary, error) {
	var hits []summarizer.ConnectionCategorySummary
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"report_type": {string(summarizer.CategorySummary)}},
	})

	if sourceId != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_id": {*sourceId}},
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
	err = client.Search(context.Background(), summarizer.ConnectionSummaryIndex, query, &response)
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
	ID      string                               `json:"_id"`
	Score   float64                              `json:"_score"`
	Index   string                               `json:"_index"`
	Type    string                               `json:"_type"`
	Version int64                                `json:"_version,omitempty"`
	Source  summarizer.ConnectionLocationSummary `json:"_source"`
	Sort    []interface{}                        `json:"sort"`
}

func FetchConnectionLocationsSummaryPage(client keibi.Client, provider source.Type, sourceID *string, sort []map[string]interface{}, size int) ([]summarizer.ConnectionLocationSummary, error) {
	var hits []summarizer.ConnectionLocationSummary
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"report_type": {string(summarizer.LocationConnectionSummary)}},
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
	err = client.Search(context.Background(), summarizer.ConnectionSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	for _, hit := range response.Hits.Hits {
		hits = append(hits, hit.Source)
	}
	return hits, nil
}

type ConnectionTrendSummaryQueryResponse struct {
	Hits ConnectionTrendSummaryQueryHits `json:"hits"`
}
type ConnectionTrendSummaryQueryHits struct {
	Total keibi.SearchTotal                `json:"total"`
	Hits  []ConnectionTrendSummaryQueryHit `json:"hits"`
}
type ConnectionTrendSummaryQueryHit struct {
	ID      string                            `json:"_id"`
	Score   float64                           `json:"_score"`
	Index   string                            `json:"_index"`
	Type    string                            `json:"_type"`
	Version int64                             `json:"_version,omitempty"`
	Source  summarizer.ConnectionTrendSummary `json:"_source"`
	Sort    []interface{}                     `json:"sort"`
}

func FetchConnectionTrendSummaryPage(client keibi.Client, sourceID *string, createdAtFrom, createdAtTo int64,
	sort []map[string]interface{}, size int) ([]summarizer.ConnectionTrendSummary, error) {
	var hits []summarizer.ConnectionTrendSummary
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"report_type": {string(summarizer.TrendConnectionSummary)}},
	})

	if sourceID != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_id": {*sourceID}},
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

	var response ConnectionTrendSummaryQueryResponse
	err = client.Search(context.Background(), summarizer.ConnectionSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	for _, hit := range response.Hits.Hits {
		hits = append(hits, hit.Source)
	}
	return hits, nil
}

type ConnectionResourceTypeTrendSummaryQueryResponse struct {
	Hits ConnectionResourceTypeTrendSummaryQueryHits `json:"hits"`
}
type ConnectionResourceTypeTrendSummaryQueryHits struct {
	Total keibi.SearchTotal                            `json:"total"`
	Hits  []ConnectionResourceTypeTrendSummaryQueryHit `json:"hits"`
}
type ConnectionResourceTypeTrendSummaryQueryHit struct {
	ID      string                                        `json:"_id"`
	Score   float64                                       `json:"_score"`
	Index   string                                        `json:"_index"`
	Type    string                                        `json:"_type"`
	Version int64                                         `json:"_version,omitempty"`
	Source  summarizer.ConnectionResourceTypeTrendSummary `json:"_source"`
	Sort    []interface{}                                 `json:"sort"`
}

func FetchConnectionResourceTypeTrendSummaryPage(client keibi.Client, sourceID *string, resourceTypes []string, createdAtFrom, createdAtTo int64,
	sort []map[string]interface{}, size int) ([]summarizer.ConnectionResourceTypeTrendSummary, error) {
	var hits []summarizer.ConnectionResourceTypeTrendSummary
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"report_type": {string(summarizer.ResourceTypeTrendConnectionSummary)}},
	})

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"resource_type": resourceTypes},
	})

	if sourceID != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_id": {*sourceID}},
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

	var response ConnectionResourceTypeTrendSummaryQueryResponse
	err = client.Search(context.Background(), summarizer.ConnectionSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	for _, hit := range response.Hits.Hits {
		hits = append(hits, hit.Source)
	}
	return hits, nil
}

type ProviderTrendSummaryQueryResponse struct {
	Hits ProviderTrendSummaryQueryHits `json:"hits"`
}
type ProviderTrendSummaryQueryHits struct {
	Total keibi.SearchTotal              `json:"total"`
	Hits  []ProviderTrendSummaryQueryHit `json:"hits"`
}
type ProviderTrendSummaryQueryHit struct {
	ID      string                          `json:"_id"`
	Score   float64                         `json:"_score"`
	Index   string                          `json:"_index"`
	Type    string                          `json:"_type"`
	Version int64                           `json:"_version,omitempty"`
	Source  summarizer.ProviderTrendSummary `json:"_source"`
	Sort    []interface{}                   `json:"sort"`
}

func FetchProviderTrendSummaryPage(client keibi.Client, provider source.Type, createdAtFrom, createdAtTo int64,
	sort []map[string]interface{}, size int) ([]summarizer.ProviderTrendSummary, error) {
	var hits []summarizer.ProviderTrendSummary
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"report_type": {string(summarizer.TrendProviderSummary)}},
	})

	if !provider.IsNull() {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_type": {provider.String()}},
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

	var response ProviderTrendSummaryQueryResponse
	err = client.Search(context.Background(), summarizer.ProviderSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	for _, hit := range response.Hits.Hits {
		hits = append(hits, hit.Source)
	}
	return hits, nil
}

type ProviderResourceTypeTrendSummaryQueryResponse struct {
	Hits ProviderResourceTypeTrendSummaryQueryHits `json:"hits"`
}
type ProviderResourceTypeTrendSummaryQueryHits struct {
	Total keibi.SearchTotal                          `json:"total"`
	Hits  []ProviderResourceTypeTrendSummaryQueryHit `json:"hits"`
}
type ProviderResourceTypeTrendSummaryQueryHit struct {
	ID      string                                      `json:"_id"`
	Score   float64                                     `json:"_score"`
	Index   string                                      `json:"_index"`
	Type    string                                      `json:"_type"`
	Version int64                                       `json:"_version,omitempty"`
	Source  summarizer.ProviderResourceTypeTrendSummary `json:"_source"`
	Sort    []interface{}                               `json:"sort"`
}

func FetchProviderResourceTypeTrendSummaryPage(client keibi.Client, provider source.Type, resourceTypes []string, createdAtFrom, createdAtTo int64,
	sort []map[string]interface{}, size int) ([]summarizer.ProviderResourceTypeTrendSummary, error) {
	var hits []summarizer.ProviderResourceTypeTrendSummary
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"report_type": {string(summarizer.ResourceTypeTrendProviderSummary)}},
	})

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"resource_type": resourceTypes},
	})

	if !provider.IsNull() {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_type": {provider.String()}},
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
	fmt.Println("query=", query)

	var response ProviderResourceTypeTrendSummaryQueryResponse
	err = client.Search(context.Background(), summarizer.ProviderSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	for _, hit := range response.Hits.Hits {
		hits = append(hits, hit.Source)
	}
	return hits, nil
}

type ProviderResourceTypeSummaryQueryResponse struct {
	Hits ProviderResourceTypeSummaryQueryHits `json:"hits"`
}
type ProviderResourceTypeSummaryQueryHits struct {
	Total keibi.SearchTotal                     `json:"total"`
	Hits  []ProviderResourceTypeSummaryQueryHit `json:"hits"`
}
type ProviderResourceTypeSummaryQueryHit struct {
	ID      string                                 `json:"_id"`
	Score   float64                                `json:"_score"`
	Index   string                                 `json:"_index"`
	Type    string                                 `json:"_type"`
	Version int64                                  `json:"_version,omitempty"`
	Source  summarizer.ProviderResourceTypeSummary `json:"_source"`
	Sort    []interface{}                          `json:"sort"`
}

func FetchProviderResourceTypeSummaryPage(client keibi.Client, provider source.Type, resourceType []string,
	sort []map[string]interface{}, size int) ([]summarizer.ProviderResourceTypeSummary, error) {
	var hits []summarizer.ProviderResourceTypeSummary
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"report_type": {string(summarizer.ResourceTypeProviderSummary)}},
	})

	if !provider.IsNull() {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_type": {provider.String()}},
		})
	}

	if resourceType != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"resource_type": resourceType},
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

	var response ProviderResourceTypeSummaryQueryResponse
	err = client.Search(context.Background(), summarizer.ProviderSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	for _, hit := range response.Hits.Hits {
		hits = append(hits, hit.Source)
	}
	return hits, nil
}

type ConnectionResourceTypeSummaryQueryResponse struct {
	Hits ConnectionResourceTypeSummaryQueryHits `json:"hits"`
}
type ConnectionResourceTypeSummaryQueryHits struct {
	Total keibi.SearchTotal                       `json:"total"`
	Hits  []ConnectionResourceTypeSummaryQueryHit `json:"hits"`
}
type ConnectionResourceTypeSummaryQueryHit struct {
	ID      string                                   `json:"_id"`
	Score   float64                                  `json:"_score"`
	Index   string                                   `json:"_index"`
	Type    string                                   `json:"_type"`
	Version int64                                    `json:"_version,omitempty"`
	Source  summarizer.ConnectionResourceTypeSummary `json:"_source"`
	Sort    []interface{}                            `json:"sort"`
}

func FetchConnectionResourceTypeSummaryPage(client keibi.Client, sourceID *string, resourceType []string,
	sort []map[string]interface{}, size int) ([]summarizer.ConnectionResourceTypeSummary, error) {
	var hits []summarizer.ConnectionResourceTypeSummary
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"report_type": {string(summarizer.ResourceTypeSummary)}},
	})

	if sourceID != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_id": {*sourceID}},
		})
	}

	if resourceType != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"resource_type": resourceType},
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

	var response ConnectionResourceTypeSummaryQueryResponse
	err = client.Search(context.Background(), summarizer.ConnectionSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	for _, hit := range response.Hits.Hits {
		hits = append(hits, hit.Source)
	}
	return hits, nil
}

type ConnectionServiceLocationsSummaryQueryResponse struct {
	Hits ConnectionServiceLocationsSummaryQueryHits `json:"hits"`
}
type ConnectionServiceLocationsSummaryQueryHits struct {
	Total keibi.SearchTotal                           `json:"total"`
	Hits  []ConnectionServiceLocationsSummaryQueryHit `json:"hits"`
}
type ConnectionServiceLocationsSummaryQueryHit struct {
	ID      string                                      `json:"_id"`
	Score   float64                                     `json:"_score"`
	Index   string                                      `json:"_index"`
	Type    string                                      `json:"_type"`
	Version int64                                       `json:"_version,omitempty"`
	Source  summarizer.ConnectionServiceLocationSummary `json:"_source"`
	Sort    []interface{}                               `json:"sort"`
}

func FetchConnectionServiceLocationsSummaryPage(client keibi.Client, provider source.Type, sourceID *string, sort []map[string]interface{}, size int) ([]summarizer.ConnectionServiceLocationSummary, error) {
	var hits []summarizer.ConnectionServiceLocationSummary
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"report_type": {string(summarizer.ServiceLocationConnectionSummary)}},
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

	var response ConnectionServiceLocationsSummaryQueryResponse
	err = client.Search(context.Background(), summarizer.ConnectionSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	for _, hit := range response.Hits.Hits {
		hits = append(hits, hit.Source)
	}
	return hits, nil
}

//{
//  "took": 234,
//  "timed_out": false,
//  "_shards": {
//    "total": 1,
//    "successful": 1,
//    "skipped": 0,
//    "failed": 0
//  },
//  "hits": {
//    "total": {
//      "value": 10000,
//      "relation": "gte"
//    },
//    "max_score": null,
//    "hits": []
//  },
//  "aggregations": {
//    "schedule_job_id_group": {
//      "doc_count_error_upper_bound": 0,
//      "sum_other_doc_count": 994296,
//      "buckets": [
//        {
//          "key": 183,
//          "doc_count": 11010,
//          "resource_type_group": {
//            "doc_count_error_upper_bound": 0,
//            "sum_other_doc_count": 0,
//            "buckets": [
//              {
//                "key": "microsoft.authorization/elevateaccessroleassignment",
//                "doc_count": 201,
//                "resource_type_count": {
//                  "value": 26497
//                }
//              }
//			  ]}}]}}}

type FetchResourceTypeCountAtTimeResponse struct {
	Aggregations struct {
		ScheduleJobIDGroup struct {
			Buckets []struct {
				ResourceTypeGroup struct {
					Buckets []struct {
						Key               string `json:"key"`
						ResourceTypeCount struct {
							Value float64 `json:"value"`
						} `json:"resource_type_count"`
					} `json:"buckets"`
				} `json:"resource_type_group"`
			} `json:"buckets"`
		} `json:"schedule_job_id_group"`
	} `json:"aggregations"`
}

func FetchResourceTypeCountAtTime(client keibi.Client, provider source.Type, sourceID *string, t time.Time, resourceTypes []string, size int) (map[string]int, error) {
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"report_type": {string(summarizer.ResourceTypeTrendConnectionSummary)}},
	})

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"resource_type": resourceTypes},
	})

	if sourceID != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_id": {*sourceID}},
		})
	}

	if !provider.IsNull() {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_type": {provider.String()}},
		})
	}

	filters = append(filters, map[string]interface{}{
		"range": map[string]interface{}{
			"described_at": map[string]string{
				"lte": strconv.FormatInt(t.UnixMilli(), 10),
			},
		},
	})

	sort := []map[string]any{
		{"_id": "desc"},
	}

	res["size"] = 0
	res["sort"] = sort
	res["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}
	res["aggs"] = map[string]any{
		"schedule_job_id_group": map[string]any{
			"terms": map[string]any{
				"field": "schedule_job_id",
				"size":  1,
				"order": map[string]string{
					"_term": "desc",
				},
			},
			"aggs": map[string]any{
				"resource_type_group": map[string]any{
					"terms": map[string]any{
						"field": "resource_type",
						"size":  size,
					},
					"aggs": map[string]any{
						"resource_type_count": map[string]any{
							"sum": map[string]any{
								"field": "resource_count",
							},
						},
					},
				},
			},
		},
	}

	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	query := string(b)
	var response FetchResourceTypeCountAtTimeResponse
	err = client.Search(context.Background(), summarizer.ConnectionSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	result := make(map[string]int)
	if len(response.Aggregations.ScheduleJobIDGroup.Buckets) == 0 {
		return result, nil
	}
	for _, bucket := range response.Aggregations.ScheduleJobIDGroup.Buckets[0].ResourceTypeGroup.Buckets {
		result[bucket.Key] = int(bucket.ResourceTypeCount.Value)
	}
	return result, nil
}

type FetchInsightValueAtTimeResponse struct {
	Aggregations struct {
		ScheduleJobIDGroup struct {
			Buckets []struct {
				ResourceTypeGroup struct {
					Buckets []struct {
						Key               int64 `json:"key"`
						ResourceTypeCount struct {
							Value float64 `json:"value"`
						} `json:"insight_values"`
					} `json:"buckets"`
				} `json:"insight_id_group"`
			} `json:"buckets"`
		} `json:"job_id_group"`
	} `json:"aggregations"`
}

func FetchInsightValueAtTime(client keibi.Client, t time.Time, insightIds []string, size int) (map[string]float64, error) {
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"resource_type": {"history"}},
	})

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"query_id": insightIds},
	})

	filters = append(filters, map[string]interface{}{
		"range": map[string]interface{}{
			"executed_at": map[string]string{
				"lte": strconv.FormatInt(t.UnixMilli(), 10),
			},
		},
	})

	sort := []map[string]any{
		{"_id": "desc"},
	}

	res["size"] = 0
	res["sort"] = sort
	res["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}
	res["aggs"] = map[string]any{
		"job_id_group": map[string]any{
			"terms": map[string]any{
				"field": "job_id",
				"size":  1,
				"order": map[string]string{
					"_term": "desc",
				},
			},
			"aggs": map[string]any{
				"insight_id_group": map[string]any{
					"terms": map[string]any{
						"field": "query_id",
						"size":  size,
					},
					"aggs": map[string]any{
						"insight_values": map[string]any{
							"sum": map[string]any{
								"field": "result",
							},
						},
					},
				},
			},
		},
	}

	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	query := string(b)
	var response FetchInsightValueAtTimeResponse
	err = client.Search(context.Background(), es.InsightsIndex, query, &response)
	if err != nil {
		return nil, err
	}

	result := make(map[string]float64)
	if len(response.Aggregations.ScheduleJobIDGroup.Buckets) == 0 {
		return result, nil
	}
	for _, bucket := range response.Aggregations.ScheduleJobIDGroup.Buckets[0].ResourceTypeGroup.Buckets {
		result[fmt.Sprintf("%d", bucket.Key)] = bucket.ResourceTypeCount.Value
	}
	return result, nil
}
