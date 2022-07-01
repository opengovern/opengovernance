package describe

import (
	"encoding/json"

	"gitlab.com/keibiengine/keibi-engine/pkg/describe/kafka"

	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
)

type CategoryQueryResponse struct {
	Hits CategoryQueryHits `json:"hits"`
}
type CategoryQueryHits struct {
	Total keibi.SearchTotal  `json:"total"`
	Hits  []CategoryQueryHit `json:"hits"`
}
type CategoryQueryHit struct {
	ID      string                      `json:"_id"`
	Score   float64                     `json:"_score"`
	Index   string                      `json:"_index"`
	Type    string                      `json:"_type"`
	Version int64                       `json:"_version,omitempty"`
	Source  kafka.SourceCategorySummary `json:"_source"`
	Sort    []interface{}               `json:"sort"`
}

func FindOldCategoryValue(jobID uint, categoryName string) (string, error) {
	boolQuery := map[string]interface{}{}
	var filters []interface{}
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"report_type": {kafka.ResourceSummaryTypeCategoryHistorySummary}},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]interface{}{"source_job_id": {jobID}},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]interface{}{"category_name": {categoryName}},
	})
	boolQuery["filter"] = filters
	res := make(map[string]interface{})
	res["size"] = 1
	res["sort"] = []map[string]interface{}{
		{
			"executed_at": "desc",
		},
	}

	if len(boolQuery) > 0 {
		res["query"] = map[string]interface{}{
			"bool": boolQuery,
		}
	}
	b, err := json.Marshal(res)
	return string(b), err
}

type ServiceQueryResponse struct {
	Hits ServiceQueryHits `json:"hits"`
}
type ServiceQueryHits struct {
	Total keibi.SearchTotal `json:"total"`
	Hits  []ServiceQueryHit `json:"hits"`
}
type ServiceQueryHit struct {
	ID      string                      `json:"_id"`
	Score   float64                     `json:"_score"`
	Index   string                      `json:"_index"`
	Type    string                      `json:"_type"`
	Version int64                       `json:"_version,omitempty"`
	Source  kafka.SourceServicesSummary `json:"_source"`
	Sort    []interface{}               `json:"sort"`
}

func FindOldServiceValue(jobID uint, categoryName string) (string, error) {
	var filters []interface{}
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"report_type": {kafka.ResourceSummaryTypeServiceHistorySummary}},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]interface{}{"source_job_id": {jobID}},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]interface{}{"service_name": {categoryName}},
	})
	res := make(map[string]interface{})
	res["size"] = 1
	res["sort"] = []map[string]interface{}{
		{
			"executed_at": "desc",
		},
	}

	res["query"] = map[string]interface{}{
		"bool": map[string]interface{}{
			"filter": filters,
		},
	}
	b, err := json.Marshal(res)
	return string(b), err
}
