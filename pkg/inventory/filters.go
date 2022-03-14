package inventory

import (
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
)

func FilterIsEmpty(filter []string) bool {
	return filter == nil || len(filter) == 0
}

func BuildBoolFilter(filters []keibi.BoolFilter) map[string]interface{} {
	return map[string]interface{}{
		"bool": map[string]interface{}{
			"filter": filters,
		},
	}
}

func BuildSort(sort Sort) []map[string]interface{} {
	var result []map[string]interface{}
	for _, item := range sort.SortBy {
		dir := string(item.Direction)
		result = append(result, map[string]interface{}{string(item.Field) + ".keyword": dir})
	}
	return result
}

func BuildQuery(shouldTerms []interface{}, size, from int, sortItems []map[string]interface{}) map[string]interface{} {
	res := map[string]interface{}{
		"size": size,
		"from": from,
	}
	if sortItems != nil {
		res["sort"] = sortItems
	}
	if shouldTerms != nil {
		res["query"] = map[string]interface{}{
			"bool": map[string]interface{}{
				"should": shouldTerms,
			},
		}
	}
	return res
}
