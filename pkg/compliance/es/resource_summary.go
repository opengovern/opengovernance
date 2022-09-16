package es

import (
	"encoding/json"
	"strconv"

	"gitlab.com/keibiengine/keibi-engine/pkg/describe/es"

	"gitlab.com/keibiengine/keibi-engine/pkg/source"

	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
)

type ComplianceTrendQueryResponse struct {
	Hits ComplianceTrendQueryHits `json:"hits"`
}
type ComplianceTrendQueryHits struct {
	Total keibi.SearchTotal         `json:"total"`
	Hits  []ComplianceTrendQueryHit `json:"hits"`
}
type ComplianceTrendQueryHit struct {
	ID      string                             `json:"_id"`
	Score   float64                            `json:"_score"`
	Index   string                             `json:"_index"`
	Type    string                             `json:"_type"`
	Version int64                              `json:"_version,omitempty"`
	Source  es.ResourceCompliancyTrendResource `json:"_source"`
	Sort    []interface{}                      `json:"sort"`
}

func FindCompliancyTrendQuery(sourceID *uuid.UUID, provider source.Type,
	describedAtFrom, describedAtTo int64, fetchSize int, searchAfter []interface{}) (string, error) {

	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"report_type": {es.ResourceSummaryTypeCompliancyTrend}},
	})

	if !provider.IsNull() {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_type": {provider.String()}},
		})
	}

	if sourceID != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_id": {sourceID.String()}},
		})
	}

	filters = append(filters, map[string]interface{}{
		"range": map[string]interface{}{
			"described_at": map[string]string{
				"gte": strconv.FormatInt(describedAtFrom, 10),
				"lte": strconv.FormatInt(describedAtTo, 10),
			},
		},
	})

	res["size"] = fetchSize
	res["sort"] = []map[string]interface{}{
		{
			"described_at": "asc",
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
