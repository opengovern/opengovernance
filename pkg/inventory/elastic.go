package inventory

import (
	"context"
	"encoding/json"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
)

func QuerySummaryResources(
	ctx context.Context,
	client keibi.Client,
	filters Filters,
	provider *SourceType,
	size, lastIndex int,
) ([]describe.KafkaLookupResource, error) {
	var err error

	var terms []keibi.BoolFilter
	if !FilterIsEmpty(filters.Location) {
		terms = append(terms, keibi.TermsFilter("location", filters.Location))
	}

	if !FilterIsEmpty(filters.ResourceType) {
		terms = append(terms, keibi.TermsFilter("resourceType", filters.ResourceType))
	}

	if !FilterIsEmpty(filters.KeibiSource) {
		terms = append(terms, keibi.TermsFilter("sourceID", filters.KeibiSource))
	}

	if provider != nil {
		terms = append(terms, keibi.TermsFilter("provider", []string{string(*provider)}))
	}

	queryStr, err := BuildSummaryQuery(terms, size, lastIndex)
	if err != nil {
		return nil, err
	}

	resources, err := SummaryQueryES(client, ctx, "inventory_summary", queryStr)
	if err != nil {
		return nil, err
	}

	return resources, nil
}

func BuildSummaryQuery(terms []keibi.BoolFilter, size, lastIdx int) (string, error) {
	if len(terms) > 0 {
		query := BuildBoolFilter(terms)
		var shouldTerms []interface{}
		shouldTerms = append(shouldTerms, query)

		query = BuildQuery(shouldTerms, size, lastIdx)
		queryBytes, err := json.Marshal(query)
		if err != nil {
			return "", err
		}

		return string(queryBytes), nil
	}
	return "", nil
}

func SummaryQueryES(client keibi.Client, ctx context.Context, index string, query string) ([]describe.KafkaLookupResource, error) {
	var response SummaryQueryResponse
	err := client.Search(ctx, index, query, &response)
	if err != nil {
		return nil, err
	}

	var resources []describe.KafkaLookupResource
	for _, hits := range response.Hits.Hits {
		resources = append(resources, hits.Source)
	}

	return resources, nil
}
