package es

import (
	"encoding/json"
	"github.com/kaytu-io/kaytu-util/pkg/es"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"golang.org/x/net/context"
)

type Response struct {
	Took     int64 `json:"took"`
	TimedOut bool  `json:"timed_out"`
	Shards   struct {
		Total      int64 `json:"total"`
		Successful int64 `json:"successful"`
		Skipped    int64 `json:"skipped"`
		Failed     int64 `json:"failed"`
	}
	Hits struct {
		Total struct {
			Value    int64  `json:"value"`
			Relation string `json:"relation"`
		}
		MaxScore int64 `json:"max_score"`
		Hits     []struct {
			Index  string `json:"_index"`
			Type   string `json:"_type"`
			Id     string `json:"_id"`
			Score  int64  `json:"_score"`
			Source struct {
				Metadata     interface{} `json:"metadata"`
				SourceJobId  int64       `json:"source_job_id"`
				ResourceType string      `json:"resource_type"`
				CreatedAt    int64       `json:"created_at"`
				Description  any         `json:"description"`
				ARN          string
				ID           string
				Name         string
				Account      string
				Region       string
				Partition    string
				Type         string
			} `json:"_source"`
		}
	}
}

func GetElasticsearch(client kaytu.Client, resourceId string, resourceType string) (*Response, error) {
	var resp Response
	index := es.ResourceTypeToESIndex(resourceType)
	queryBytes, err := GetResourceQuery(resourceId)
	if err != nil {
		return nil, err
	}
	err = client.Search(context.Background(), index, string(queryBytes), &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

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
