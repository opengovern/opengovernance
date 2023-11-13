package es

import (
	"encoding/json"
	"github.com/kaytu-io/kaytu-util/pkg/es"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"go.uber.org/zap"
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
		MaxScore float64 `json:"max_score"`
		Hits     []struct {
			Index  string  `json:"_index"`
			Type   string  `json:"_type"`
			Id     string  `json:"_id"`
			Score  float64 `json:"_score"`
			Source struct {
				Metadata struct {
					Partition    string `json:"Partition"`
					AccountID    string `json:"AccountID"`
					SourceID     string `json:"SourceID"`
					name         string `json:"name"`
					Region       string `json:"Region"`
					ResourceType string `json:"ResourceType"`
					Name         string `json:"Name"`
				} `json:"metadata"`
				SourceJobId   int64  `json:"source_job_id"`
				EsId          string `json:"es_id"`
				ResourceType  string `json:"resource_type"`
				CreatedAt     int64  `json:"created_at"`
				Description   any    `json:"description"`
				ResourceJobId int64  `json:"resource_job_id"`
				ResourceGroup string `json:"resource_group"`
				Name          string `json:"name"`
				Location      string `json:"location"`
				EsIndex       string `json:"es_index"`
				ID            string `json:"id"`
				ScheduleJobId string `json:"schedule_job_id"`
				SourceId      string `json:"source_id"`
				ARN           string `json:"arn"`
			} `json:"_source"`
		}
	}
}

func GetElasticsearch(logger *zap.Logger, client kaytu.Client, resourceId string, resourceType string) (*Response, error) {
	var resp Response
	index := es.ResourceTypeToESIndex(resourceType)
	queryBytes, err := GetResourceQuery(logger, resourceId)
	if err != nil {
		return nil, err
	}
	err = client.Search(context.Background(), index, string(queryBytes), &resp)
	if err != nil {
		logger.Error("Error getting Elastic response", zap.Error(err))
		return nil, err
	}
	return &resp, nil
}

func GetResourceQuery(logger *zap.Logger, resourceId string) ([]byte, error) {
	//terms := make(map[string]any)
	//terms["id"] = resourceId
	//
	//root := map[string]any{}
	//
	//boolQuery := make(map[string]any)
	//if terms != nil && len(terms) > 0 {
	//	var filters []map[string]any
	//	for k, vs := range terms {
	//		filters = append(filters, map[string]any{
	//			"terms": map[string]any{
	//				k: vs,
	//			},
	//		})
	//	}
	//
	//	boolQuery["filter"] = filters
	//}
	//if len(boolQuery) > 0 {
	//	root["query"] = map[string]any{
	//		"bool": boolQuery,
	//	}
	//}

	root := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"filter": []map[string]any{
					{
						"term": map[string]string{
							"id": resourceId,
						},
					},
				},
			},
		},
	}
	queryBytes, err := json.Marshal(root)
	if err != nil {
		logger.Error("Unable to marshal", zap.Error(err))
		return nil, err
	}

	return queryBytes, nil
}
