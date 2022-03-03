package inventory

import "gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"

func FilterIsEmpty(filter []string) bool {
	return filter == nil || len(filter) == 0
}

func FilterContains(filters []string, obj string) bool {
	for _, f := range filters {
		if f == obj {
			return true
		}
	}
	return false
}

func BuildBoolFilter(filters []keibi.BoolFilter) map[string]interface{} {
	return map[string]interface{}{
		"bool": map[string]interface{}{
			"filter": filters,
		},
	}
}

func BuildQuery(shouldTerms []interface{}, size, from int) map[string]interface{} {
	return map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"should": shouldTerms,
			},
		},
		"size": size,
		"from": from,
	}
}