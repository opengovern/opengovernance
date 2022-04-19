package api

import (
	"context"
	"encoding/json"

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
) ([]describe.KafkaLookupResource, error) {
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

	queryStr, err := BuildSummaryQuery(query, terms, size, lastIndex, sorts)
	if err != nil {
		return nil, err
	}

	resources, err := SummaryQueryES(client, ctx, describe.InventorySummaryIndex, queryStr)
	if err != nil {
		return nil, err
	}

	return resources, nil
}

func BuildSummaryQuery(query string, terms map[string][]string, size, lastIdx int, sorts []ResourceSortItem) (string, error) {
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
