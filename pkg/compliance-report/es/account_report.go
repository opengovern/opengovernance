package es

import (
	"encoding/json"

	"gitlab.com/keibiengine/keibi-engine/pkg/source"
)

type AccountReportType string

const (
	AccountReportTypeLast   = "last"
	AccountReportTypeInTime = "intime"
)

func ComplianceScoreByProviderQuery(provider source.Type, sourceID *string, size int, order string, searchAfter []interface{}) (string, error) {
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"provider": {string(provider)}},
	})
	if sourceID != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"sourceID": {string(*sourceID)}},
		})
	}
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]interface{}{"accountReportType": {AccountReportTypeLast}},
	})

	res["size"] = size
	res["sort"] = []map[string]interface{}{
		{
			"compliancePercentage": order,
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
