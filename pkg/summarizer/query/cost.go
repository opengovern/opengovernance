package query

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	describe "gitlab.com/keibiengine/keibi-engine/pkg/describe/es"
	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
	"gitlab.com/keibiengine/keibi-engine/pkg/types"
)

type ResourceQueryResponse struct {
	Hits ResourceQueryHits `json:"hits"`
}
type ResourceQueryHits struct {
	Total keibi.SearchTotal  `json:"total"`
	Hits  []ResourceQueryHit `json:"hits"`
}
type ResourceQueryHit struct {
	ID      string            `json:"_id"`
	Score   float64           `json:"_score"`
	Index   string            `json:"_index"`
	Type    string            `json:"_type"`
	Version int64             `json:"_version,omitempty"`
	Source  describe.Resource `json:"_source"`
	Sort    []interface{}     `json:"sort"`
}

func GetResourceFromResourceLookup(client keibi.Client, resource describe.LookupResource) (*describe.Resource, error) {
	res := make(map[string]interface{})
	var filters []interface{}

	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"source_job_id": {strconv.Itoa(int(resource.SourceJobID))}},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"resource_type": {strings.ToLower(resource.ResourceType)}},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"resource_job_id": {strconv.Itoa(int(resource.ResourceJobID))}},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"id": {resource.ResourceID}},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"source_id": {resource.SourceID}},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]string{"source_type": {resource.SourceType.String()}},
	})

	sort := []map[string]interface{}{{"_id": "desc"}}
	res["sort"] = sort
	res["query"] = map[string]interface{}{
		"bool": map[string]interface{}{
			"filter": filters,
		},
	}
	bytes, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	query := string(bytes)
	fmt.Println("query=", query)

	var response ResourceQueryResponse
	err = client.Search(context.Background(), types.ResourceTypeToESIndex(resource.ResourceType), query, &response)
	if err != nil {
		return nil, err
	}
	if len(response.Hits.Hits) == 0 {
		return nil, fmt.Errorf("no resource found")
	}
	return &response.Hits.Hits[0].Source, nil
}
