package es

import (
	"encoding/json"
	"strconv"

	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"

	"gitlab.com/keibiengine/keibi-engine/pkg/source"
)

const (
	esIndexHeader          = "elasticsearch_index"
	CompliancySummaryIndex = "compliance_summary"
)

type CompliancySummaryType string

const (
	CompliancySummaryTypeServiceSummary CompliancySummaryType = "service"
)

type CompliancySummary struct {
	CompliancySummaryType CompliancySummaryType `json:"compliancySummaryType"`
	ReportJobId           uint                  `json:"reportJobID"`
	Provider              source.Type           `json:"provider"`
	DescribedAt           int64                 `json:"describedAt"`
}

func (r CompliancySummary) KeysAndIndex() ([]string, string) {
	return []string{
		string(r.CompliancySummaryType),
		strconv.FormatInt(int64(r.ReportJobId), 10),
	}, CompliancySummaryIndex
}

type ServiceCompliancySummary struct {
	ServiceName string `json:"serviceName"`

	TotalResources       int     `json:"totalResources"`
	TotalCompliant       int     `json:"totalCompliant"`
	CompliancePercentage float64 `json:"compliancePercentage"`

	CompliancySummary
}

func (r ServiceCompliancySummary) KeysAndIndex() ([]string, string) {
	return []string{
		r.ServiceName,
		string(r.CompliancySummaryType),
		strconv.FormatInt(int64(r.ReportJobId), 10),
	}, CompliancySummaryIndex
}

type ServiceCompliancySummaryQueryResponse struct {
	Hits ServiceCompliancySummaryQueryHits `json:"hits"`
}
type ServiceCompliancySummaryQueryHits struct {
	Total keibi.SearchTotal                  `json:"total"`
	Hits  []ServiceCompliancySummaryQueryHit `json:"hits"`
}
type ServiceCompliancySummaryQueryHit struct {
	ID      string                   `json:"_id"`
	Score   float64                  `json:"_score"`
	Index   string                   `json:"_index"`
	Type    string                   `json:"_type"`
	Version int64                    `json:"_version,omitempty"`
	Source  ServiceCompliancySummary `json:"_source"`
	Sort    []interface{}            `json:"sort"`
}

func ServiceComplianceScoreByProviderQuery(provider source.Type, size int, order string, searchAfter []interface{}) (string, error) {
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"provider": {string(provider)}},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]interface{}{"compliancySummaryType": {CompliancySummaryTypeServiceSummary}},
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
