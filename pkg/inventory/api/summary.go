package api

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kaytu-io/kaytu-engine/pkg/describe/es"
	"github.com/kaytu-io/kaytu-util/pkg/source"

	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
)

const InventorySummaryIndex = "inventory_summary"

func QuerySummaryResources(
	ctx context.Context,
	client keibi.Client,
	query string,
	filters Filters,
	connector []source.Type,
	size, lastIndex int,
	sorts []ResourceSortItem,
) ([]es.LookupResource, keibi.SearchTotal, error) {
	var err error

	terms := make(map[string][]string)
	if !FilterIsEmpty(filters.Location) {
		terms["location"] = filters.Location
	}

	if !FilterIsEmpty(filters.ResourceType) {
		terms["resource_type"] = filters.ResourceType
	}

	if !FilterIsEmpty(filters.ConnectionID) {
		terms["source_id"] = filters.ConnectionID
	}

	if len(connector) > 0 {
		connectorStrs := make([]string, 0, len(connector))
		for _, c := range connector {
			connectorStrs = append(connectorStrs, c.String())
		}
		terms["source_type"] = connectorStrs
	}

	queryStr, err := BuildSummaryQuery(query, terms, nil, size, lastIndex, sorts)
	if err != nil {
		return nil, keibi.SearchTotal{}, err
	}

	resources, resultCount, err := SummaryQueryES(client, ctx, InventorySummaryIndex, queryStr)
	if err != nil {
		return nil, keibi.SearchTotal{}, err
	}

	return resources, resultCount, nil
}

func BuildSummaryQuery(query string, terms map[string][]string, notTerms map[string][]string, size, lastIdx int, sorts []ResourceSortItem) (string, error) {
	q := map[string]any{
		"size": size,
		"from": lastIdx,
	}
	if sorts != nil && len(sorts) > 0 {
		q["sort"] = BuildSort(sorts)
	}

	boolQuery := make(map[string]any)
	if terms != nil && len(terms) > 0 {
		var filters []map[string]any
		for k, vs := range terms {
			filters = append(filters, map[string]any{
				"terms": map[string][]string{
					k: vs,
				},
			})
		}

		boolQuery["filter"] = filters
	}
	if len(query) > 0 {
		boolQuery["must"] = map[string]any{
			"multi_match": map[string]any{
				"fields": []string{"resource_id", "source_type", "resource_type", "resource_group",
					"location", "source_id"},
				"query":     query,
				"fuzziness": "AUTO",
			},
		}
	}
	if len(notTerms) > 0 {
		var filters []map[string]any
		for k, vs := range notTerms {
			filters = append(filters, map[string]any{
				"terms": map[string][]string{
					k: vs,
				},
			})
		}

		boolQuery["must_not"] = filters
	}
	if len(boolQuery) > 0 {
		q["query"] = map[string]any{
			"bool": boolQuery,
		}
	}

	queryBytes, err := json.Marshal(q)
	if err != nil {
		return "", err
	}
	return string(queryBytes), nil
}

func SummaryQueryES(client keibi.Client, ctx context.Context, index string, query string) ([]es.LookupResource, keibi.SearchTotal, error) {
	var response SummaryQueryResponse

	fmt.Println("query:", query, "index:", index)

	err := client.SearchWithTrackTotalHits(ctx, index, query, &response, true)
	if err != nil {
		return nil, keibi.SearchTotal{}, err
	}

	var resources []es.LookupResource
	for _, hits := range response.Hits.Hits {
		resources = append(resources, hits.Source)
	}

	return resources, response.Hits.Total, nil
}
