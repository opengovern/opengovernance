package es

import (
	"encoding/json"
)

func GetResourceQuery(resourceId string) ([]byte, error) {
	terms := make(map[string]any)
	terms["id"] = resourceId

	root := map[string]any{}

	boolQuery := make(map[string]any)
	if terms != nil && len(terms) > 0 {
		var filters []map[string]any
		for k, vs := range terms {
			filters = append(filters, map[string]any{
				"terms": map[string]any{
					k: vs,
				},
			})
		}

		boolQuery["filter"] = filters
	}
	if len(boolQuery) > 0 {
		root["query"] = map[string]any{
			"bool": boolQuery,
		}
	}

	queryBytes, err := json.Marshal(root)
	if err != nil {
		return nil, err
	}

	return queryBytes, nil
}
