package api

import (
	"context"
	"encoding/json"
	"fmt"

	"gitlab.com/keibiengine/keibi-engine/pkg/describe/kafka"

	"gitlab.com/keibiengine/keibi-engine/pkg/describe"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
)

func QuerySummaryResources(
	ctx context.Context,
	client keibi.Client,
	query string,
	filters Filters,
	provider *SourceType,
	size, lastIndex int,
	sorts []ResourceSortItem,
	commonFilter *bool,
) ([]kafka.LookupResource, keibi.SearchTotal, error) {
	var err error

	terms := make(map[string][]string)
	if !FilterIsEmpty(filters.Location) {
		terms["location.keyword"] = filters.Location
	}

	if !FilterIsEmpty(filters.ResourceType) {
		terms["resource_type.keyword"] = filters.ResourceType
	}

	if !FilterIsEmpty(filters.SourceID) {
		terms["source_id.keyword"] = filters.SourceID
	}

	if provider != nil {
		terms["source_type.keyword"] = []string{string(*provider)}
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
	notTerms["resource_type.keyword"] = ignoreResourceTypes

	queryStr, err := BuildSummaryQuery(query, terms, notTerms, size, lastIndex, sorts)
	if err != nil {
		return nil, keibi.SearchTotal{}, err
	}

	resources, resultCount, err := SummaryQueryES(client, ctx, describe.InventorySummaryIndex, queryStr)
	if err != nil {
		return nil, keibi.SearchTotal{}, err
	}

	return resources, resultCount, nil
}

func BuildSummaryQuery(query string, terms map[string][]string, notTerms map[string][]string, size, lastIdx int, sorts []ResourceSortItem) (string, error) {
	q := map[string]interface{}{
		"size": size,
		"from": lastIdx,
	}
	if sorts != nil && len(sorts) > 0 {
		q["sort"] = BuildSort(sorts)
	}

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
		q["query"] = map[string]interface{}{
			"bool": boolQuery,
		}
	}

	queryBytes, err := json.Marshal(q)
	if err != nil {
		return "", err
	}
	return string(queryBytes), nil
}

func SummaryQueryES(client keibi.Client, ctx context.Context, index string, query string) ([]kafka.LookupResource, keibi.SearchTotal, error) {
	var response SummaryQueryResponse
	err := client.SearchWithTrackTotalHits(ctx, index, query, &response, true)
	if err != nil {
		return nil, keibi.SearchTotal{}, err
	}

	var resources []kafka.LookupResource
	for _, hits := range response.Hits.Hits {
		resources = append(resources, hits.Source)
	}

	return resources, response.Hits.Total, nil
}
