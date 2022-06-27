package insight

import (
	"encoding/json"

	"gitlab.com/keibiengine/keibi-engine/pkg/insight/kafka"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
)

type ResultQueryResponse struct {
	Hits ResultQueryHits `json:"hits"`
}
type ResultQueryHits struct {
	Total keibi.SearchTotal `json:"total"`
	Hits  []ResultQueryHit  `json:"hits"`
}
type ResultQueryHit struct {
	ID      string                `json:"_id"`
	Score   float64               `json:"_score"`
	Index   string                `json:"_index"`
	Type    string                `json:"_type"`
	Version int64                 `json:"_version,omitempty"`
	Source  kafka.InsightResource `json:"_source"`
	Sort    []interface{}         `json:"sort"`
}

func FindOldInsightValue(jobID, queryID uint) (string, error) {
	boolQuery := map[string]interface{}{}
	var filters []interface{}
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"resource_type": {kafka.InsightResourceHistory}},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]interface{}{"job_id": {jobID}},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]interface{}{"query_id": {queryID}},
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
