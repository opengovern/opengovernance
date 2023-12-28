package es

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	"github.com/kaytu-io/kaytu-engine/pkg/types"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"go.uber.org/zap"
)

type ResourceFindingsQueryHit struct {
	ID      string                `json:"_id"`
	Score   float64               `json:"_score"`
	Index   string                `json:"_index"`
	Type    string                `json:"_type"`
	Version int64                 `json:"_version,omitempty"`
	Source  types.ResourceFinding `json:"_source"`
	Sort    []any                 `json:"sort"`
}

type ResourceFindingsQueryResponse struct {
	Hits struct {
		Total kaytu.SearchTotal          `json:"total"`
		Hits  []ResourceFindingsQueryHit `json:"hits"`
	} `json:"hits"`
}

func DeleteOtherResourceFindingsExcept(logger *zap.Logger, client kaytu.Client, kaytuResourceIDs []string, lessThanJobId uint) error {
	queryMap := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must_not": map[string]any{
					"terms": map[string]any{
						"kaytuResourceID": kaytuResourceIDs,
					},
				},
				"filter": map[string]any{
					"range": map[string]any{
						"jobId": map[string]any{
							"lt": lessThanJobId,
						},
					},
				},
			},
		},
	}
	if len(kaytuResourceIDs) == 0 {
		delete(queryMap["query"].(map[string]any)["bool"].(map[string]any), "must_not")
	}

	query, err := json.Marshal(queryMap)
	if err != nil {
		logger.Error("failed to marshal query", zap.Error(err))
		return err
	}

	es := client.ES()
	_, err = es.DeleteByQuery(
		[]string{types.ResourceFindingsIndex},
		bytes.NewReader(query),
		es.DeleteByQuery.WithContext(context.TODO()),
	)
	if err != nil {
		logger.Error("failed to delete old resource findings", zap.Error(err), zap.String("query", string(query)), zap.Uint("lessThanJobId", lessThanJobId))
		return err
	}

	return nil
}

func ResourceFindingsQuery(logger *zap.Logger, client kaytu.Client,
	connector []source.Type, connectionID []string, resourceCollection []string,
	resourceTypes []string,
	benchmarkID []string, controlID []string,
	severity []types.FindingSeverity, conformanceStatuses []types.ConformanceStatus,
	sorts []api.ResourceFindingsSort, pageSizeLimit int, searchAfter []any) ([]ResourceFindingsQueryHit, int64, error) {

	nestedFilters := make([]map[string]any, 0)
	if len(connector) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string]any{
				"findings.connector": connector,
			},
		})
	}
	if len(connectionID) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string]any{
				"findings.connectionID": connectionID,
			},
		})
	}
	if len(benchmarkID) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string]any{
				"findings.benchmarkID": benchmarkID,
			},
		})
	}
	if len(controlID) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string]any{
				"findings.controlID": controlID,
			},
		})
	}
	if len(severity) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string]any{
				"findings.severity": severity,
			},
		})
	}
	if len(conformanceStatuses) > 0 {
		nestedFilters = append(nestedFilters, map[string]any{
			"terms": map[string]any{
				"findings.conformanceStatus": conformanceStatuses,
			},
		})
	}

	filters := make([]map[string]any, 0)
	if len(resourceTypes) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string]any{
				"resourceType": utils.ToLowerStringSlice(resourceTypes),
			},
		})
	}
	if len(resourceCollection) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string]any{
				"resourceCollection": resourceCollection,
			},
		})
	}

	requestMap := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"filter": filters,
			},
			"nested": map[string]any{
				"path": "findings",
				"query": map[string]any{
					"bool": map[string]any{
						"filter": nestedFilters,
					},
				},
			},
		},
		"size": pageSizeLimit,
	}
	if len(filters) == 0 {
		delete(requestMap["query"].(map[string]any), "bool")
	}
	if len(nestedFilters) == 0 {
		delete(requestMap["query"].(map[string]any), "nested")
	}
	if len(requestMap["query"].(map[string]any)) == 0 {
		delete(requestMap, "query")
	}
	if len(searchAfter) > 0 {
		requestMap["search_after"] = searchAfter
	}

	requestSort := make([]map[string]any, 0, len(sorts)+1)
	for _, sort := range sorts {
		switch {
		case sort.KaytuResourceID != nil:
			requestSort = append(requestSort, map[string]any{
				"kaytuResourceID": *sort.KaytuResourceID,
			})
		case sort.ResourceType != nil:
			requestSort = append(requestSort, map[string]any{
				"resourceType": *sort.ResourceType,
			})
		case sort.ResourceName != nil:
			requestSort = append(requestSort, map[string]any{
				"resourceName": *sort.ResourceName,
			})
		case sort.ResourceLocation != nil:
			requestSort = append(requestSort, map[string]any{
				"resourceLocation": *sort.ResourceLocation,
			})
		case sort.FailedCount != nil:
			scriptSource :=
				fmt.Sprintf(`int total = 0; 
for (int i=0; i<params['_source']['findings'].length;++i) { 
  if(params['_source']['findings'][i]['conformanceStatus'] != '%s') 
    total+=1;
  } 
return total;`, types.ConformanceStatusOK)
			requestSort = append(requestSort, map[string]any{
				"_script": map[string]any{
					"type": "number",
					"script": map[string]any{
						"lang":   "painless",
						"source": scriptSource,
					},
					"order": *sort.FailedCount,
				},
			})
		}
	}
	requestSort = append(requestSort, map[string]any{
		"_id": "asc",
	})
	requestMap["sort"] = requestSort

	request, err := json.Marshal(requestMap)
	if err != nil {
		logger.Error("resourceFindingsQuery - failed to marshal request", zap.Error(err), zap.Any("request", requestMap))
		return nil, 0, err
	}
	logger.Info("ResourceFindingsQuery", zap.String("request", string(request)), zap.String("index", types.ResourceFindingsIndex))

	var response ResourceFindingsQueryResponse
	err = client.SearchWithTrackTotalHits(context.Background(), types.ResourceFindingsIndex, string(request), nil, &response, true)
	if err != nil {
		return nil, 0, err
	}

	return response.Hits.Hits, response.Hits.Total.Value, nil
}
