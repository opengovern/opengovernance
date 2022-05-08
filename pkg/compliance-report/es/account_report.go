package es

import (
	"encoding/json"

	compliance_report "gitlab.com/keibiengine/keibi-engine/pkg/compliance-report"
	"gitlab.com/keibiengine/keibi-engine/pkg/utils"
)

func ComplianceScoreByProviderQuery(provider utils.SourceType, size int, searchAfter []interface{}) (string, error) {
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"provider": {string(provider)}},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]interface{}{"accountReportType": {compliance_report.AccountReportTypeLast}},
	})

	res["size"] = size
	res["sort"] = []map[string]interface{}{
		{
			"compliancePercentage": "desc",
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
