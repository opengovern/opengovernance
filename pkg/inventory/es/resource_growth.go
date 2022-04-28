package es

import (
	"encoding/json"
	"strconv"

	"github.com/google/uuid"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
)

type ResourceGrowth struct {
}

type ResourceGrowthQueryResponse struct {
	Hits ResourceGrowthQueryHits `json:"hits"`
}
type ResourceGrowthQueryHits struct {
	Total keibi.SearchTotal        `json:"total"`
	Hits  []ResourceGrowthQueryHit `json:"hits"`
}
type ResourceGrowthQueryHit struct {
	ID      string                               `json:"_id"`
	Score   float64                              `json:"_score"`
	Index   string                               `json:"_index"`
	Type    string                               `json:"_type"`
	Version int64                                `json:"_version,omitempty"`
	Source  describe.KafkaSourceResourcesSummary `json:"_source"`
	Sort    []interface{}                        `json:"sort"`
}

func (r ResourceGrowth) FindDataPointsQuery(sourceID *uuid.UUID, provider *string,
	createdAtFrom, createdAtTo int64, fetchSize int, searchAfter []interface{}) (string, error) {

	res := make(map[string]interface{})
	var filters []interface{}

	if provider != nil {
		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_type": {*provider}},
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
				"gte": strconv.FormatInt(createdAtFrom, 10),
				"lte": strconv.FormatInt(createdAtTo, 10),
			},
		},
	})

	res["size"] = fetchSize
	res["sort"] = []map[string]interface{}{
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
