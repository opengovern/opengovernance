package es

import (
	"context"
	"encoding/json"
	"fmt"

	"gitlab.com/keibiengine/keibi-engine/pkg/describe"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/kafka"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
)

var EsFetchPageSize = 10000

type ResourceSummaryQueryResponse struct {
	Hits ResourceSummaryQueryHits `json:"hits"`
}
type ResourceSummaryQueryHits struct {
	Total keibi.SearchTotal         `json:"total"`
	Hits  []ResourceSummaryQueryHit `json:"hits"`
}
type ResourceSummaryQueryHit struct {
	ID      string                       `json:"_id"`
	Score   float64                      `json:"_score"`
	Index   string                       `json:"_index"`
	Type    string                       `json:"_type"`
	Version int64                        `json:"_version,omitempty"`
	Source  kafka.SourceResourcesSummary `json:"_source"`
	Sort    []interface{}                `json:"sort"`
}

func FetchResourceSummary(client keibi.Client, jobID uint, provider *string, sourceID *string, resourceType *string) ([]kafka.SourceResourcesSummary, error) {
	var hits []kafka.SourceResourcesSummary
	var searchAfter []interface{}
	for {
		res := make(map[string]interface{})
		var filters []interface{}

		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"report_type": {kafka.ResourceSummaryTypeLastSummary}},
		})

		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_job_id": {fmt.Sprintf("%d", jobID)}},
		})

		if provider != nil {
			filters = append(filters, map[string]interface{}{
				"terms": map[string][]string{"source_type": {*provider}},
			})
		}

		if sourceID != nil {
			filters = append(filters, map[string]interface{}{
				"terms": map[string][]string{"source_id": {*sourceID}},
			})
		}

		if resourceType != nil {
			filters = append(filters, map[string]interface{}{
				"terms": map[string][]string{"resource_type": {*resourceType}},
			})
		}

		if searchAfter != nil {
			res["search_after"] = searchAfter
		}

		res["size"] = EsFetchPageSize
		res["sort"] = []map[string]interface{}{
			{
				"_id": "desc",
			},
		}
		res["query"] = map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": filters,
			},
		}
		b, err := json.Marshal(res)
		if err != nil {
			return nil, err
		}

		query := string(b)

		var response ResourceSummaryQueryResponse
		err = client.Search(context.Background(), describe.SourceResourcesSummary, query, &response)
		if err != nil {
			return nil, err
		}

		if len(response.Hits.Hits) == 0 {
			break
		}

		for _, hit := range response.Hits.Hits {
			searchAfter = hit.Sort
			hits = append(hits, hit.Source)
		}
	}
	return hits, nil
}
