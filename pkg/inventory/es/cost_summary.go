package es

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kaytu-io/kaytu-aws-describer/aws/model"
	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	summarizer "gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"
)

type FetchCostByServicesQueryResponse struct {
	Aggregation struct {
		ServiceGroup struct {
			Buckets []struct {
				Key                string `json:"key"`
				EndTimeAggregation struct {
					Hits struct {
						Total keibi.SearchTotal `json:"total"`
						Hits  []struct {
							ID      string                        `json:"_id"`
							Score   float64                       `json:"_score"`
							Index   string                        `json:"_index"`
							Type    string                        `json:"_type"`
							Version int64                         `json:"_version,omitempty"`
							Source  summarizer.ServiceCostSummary `json:"_source"`
						} `json:"hits"`
					} `json:"hits"`
				} `json:"period_end_max"`
			} `json:"buckets"`
		} `json:"service_grouping"`
	} `json:"aggregations"`
}

func FetchCostByServicesBetween(client keibi.Client, connectionIDs []string, connectors []source.Type, services []string, before time.Time, after time.Time, size int) (map[string]summarizer.ServiceCostSummary, error) {
	before = before.Truncate(24 * time.Hour)
	after = after.Truncate(24 * time.Hour)

	hits := make(map[string]summarizer.ServiceCostSummary)
	res := make(map[string]any)
	var filters []any

	filters = append(filters, map[string]any{
		"terms": map[string][]string{"report_type": {string(summarizer.CostProviderSummaryMonthly)}},
	})
	filters = append(filters, map[string]any{
		"terms": map[string][]string{"service_name": services},
	})
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"period_end": map[string]string{
				"gte": strconv.FormatInt(after.Unix(), 10),
				"lte": strconv.FormatInt(before.Unix(), 10),
			},
		},
	})

	if len(connectionIDs) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"source_id": connectionIDs},
		})
	}
	if len(connectors) > 0 {
		connectorsStr := make([]string, 0, len(connectors))
		for _, connector := range connectors {
			connectorsStr = append(connectorsStr, string(connector))
		}
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"source_type": connectorsStr},
		})
	}

	res["size"] = size
	res["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}
	res["aggs"] = map[string]any{
		"service_grouping": map[string]any{
			"terms": map[string]any{
				"field": "service_name",
			},
			"aggs": map[string]any{
				"period_end_max": map[string]any{
					"top_hits": map[string]any{
						"size": 1,
						"sort": []map[string]any{
							{
								"period_end": map[string]any{
									"order": "desc",
								},
							},
						},
					},
				},
			},
		},
	}
	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	query := string(b)
	fmt.Println("query=", query)

	var response FetchCostByServicesQueryResponse
	err = client.Search(context.Background(), summarizer.CostSummeryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	for _, bucket := range response.Aggregation.ServiceGroup.Buckets {
		for _, hit := range bucket.EndTimeAggregation.Hits.Hits {
			hits[hit.Source.ServiceName] = hit.Source
		}
	}

	return hits, nil
}

type FetchCostHistoryByServicesQueryResponse struct {
	Hits struct {
		Total keibi.SearchTotal `json:"total"`
		Hits  []struct {
			ID      string                        `json:"_id"`
			Score   float64                       `json:"_score"`
			Index   string                        `json:"_index"`
			Type    string                        `json:"_type"`
			Version int64                         `json:"_version,omitempty"`
			Source  summarizer.ServiceCostSummary `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

func FetchCostHistoryByServicesBetween(client keibi.Client, sourceID *string, provider *source.Type, services []string, before time.Time, after time.Time, size int) (map[string][]summarizer.ServiceCostSummary, error) {
	before = before.Truncate(24 * time.Hour)
	after = after.Truncate(24 * time.Hour)

	hits := make(map[string][]summarizer.ServiceCostSummary)
	res := make(map[string]any)
	var filters []any

	filters = append(filters, map[string]any{
		"terms": map[string][]string{"report_type": {string(summarizer.CostProviderSummaryMonthly)}},
	})
	filters = append(filters, map[string]any{
		"terms": map[string][]string{"service_name": services},
	})
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"period_end": map[string]string{
				"gte": strconv.FormatInt(after.Unix(), 10),
				"lte": strconv.FormatInt(before.Unix(), 10),
			},
		},
	})

	if sourceID != nil {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"source_id": {*sourceID}},
		})
	}
	if provider != nil && !provider.IsNull() {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"source_type": {(*provider).String()}},
		})
	}

	res["size"] = size
	res["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}

	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	query := string(b)
	fmt.Println("query=", query)

	var response FetchCostHistoryByServicesQueryResponse
	err = client.Search(context.Background(), summarizer.CostSummeryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	for _, hit := range response.Hits.Hits {
		if v, ok := hits[hit.Source.ServiceName]; !ok {
			hits[hit.Source.ServiceName] = []summarizer.ServiceCostSummary{
				hit.Source,
			}
		} else {
			hits[hit.Source.ServiceName] = append(v, hit.Source)
		}
	}

	return hits, nil
}

type FetchCostByAccountsQueryResponse struct {
	Aggregation struct {
		SourceIDGroup struct {
			Buckets []struct {
				Key                string `json:"key"`
				EndTimeAggregation struct {
					Hits struct {
						Total keibi.SearchTotal `json:"total"`
						Hits  []struct {
							ID      string                           `json:"_id"`
							Score   float64                          `json:"_score"`
							Index   string                           `json:"_index"`
							Type    string                           `json:"_type"`
							Version int64                            `json:"_version,omitempty"`
							Source  summarizer.ConnectionCostSummary `json:"_source"`
						} `json:"hits"`
					} `json:"hits"`
				} `json:"period_end_max"`
			} `json:"buckets"`
		} `json:"source_id_grouping"`
	} `json:"aggregations"`
}

func FetchCostByAccountsBetween(client keibi.Client, sourceID *string, provider *source.Type, before time.Time, after time.Time, size int) (map[string]summarizer.ConnectionCostSummary, error) {
	before = before.Truncate(24 * time.Hour)
	after = after.Truncate(24 * time.Hour)

	hits := make(map[string]summarizer.ConnectionCostSummary)
	res := make(map[string]any)
	var filters []any

	filters = append(filters, map[string]any{
		"terms": map[string][]string{"report_type": {string(summarizer.CostConnectionSummaryMonthly)}},
	})
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"period_end": map[string]string{
				"gte": strconv.FormatInt(after.Unix(), 10),
				"lte": strconv.FormatInt(before.Unix(), 10),
			},
		},
	})

	if sourceID != nil {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"source_id": {*sourceID}},
		})
	}
	if provider != nil && !provider.IsNull() {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"source_type": {(*provider).String()}},
		})
	}

	res["size"] = size
	res["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}
	res["aggs"] = map[string]any{
		"source_id_grouping": map[string]any{
			"terms": map[string]string{
				"field": "source_id",
			},
			"aggs": map[string]any{
				"period_end_max": map[string]any{
					"top_hits": map[string]any{
						"size": 1,
						"sort": []map[string]string{
							{
								"period_end": "desc",
							},
						},
					},
				},
			},
		},
	}

	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	query := string(b)
	fmt.Println("query=", query, "index=", summarizer.CostSummeryIndex)

	var response FetchCostByAccountsQueryResponse
	err = client.Search(context.Background(), summarizer.CostSummeryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	for _, bucket := range response.Aggregation.SourceIDGroup.Buckets {
		for _, hit := range bucket.EndTimeAggregation.Hits.Hits {
			hits[bucket.Key] = hit.Source
		}
	}

	for _, hit := range hits {
		switch strings.ToLower(hit.ResourceType) {
		case "aws::costexplorer::byaccountmonthly":
			hitCostStr, err := json.Marshal(hit.Cost)
			if err != nil {
				return nil, err
			}
			var hitCost model.CostExplorerByAccountMonthlyDescription
			err = json.Unmarshal(hitCostStr, &hitCost)
			if err != nil {
				return nil, err
			}
			hit.Cost = hitCost
		}
	}

	return hits, nil
}

func FetchDailyCostHistoryByServicesBetween(client keibi.Client, connectionIDs []string, connectors []source.Type, services []string, before time.Time, after time.Time, size int) (map[string][]summarizer.ServiceCostSummary, error) {
	before = before.Truncate(24 * time.Hour)
	after = after.Truncate(24 * time.Hour)

	hits := make(map[string][]summarizer.ServiceCostSummary)
	res := make(map[string]any)
	var filters []any

	filters = append(filters, map[string]any{
		"terms": map[string][]string{"report_type": {string(summarizer.CostProviderSummaryDaily)}},
	})
	if len(services) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"service_name": services},
		})
	}
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"period_end": map[string]string{
				"lte": strconv.FormatInt(before.Unix(), 10),
			},
		},
	})
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"period_start": map[string]string{
				"gte": strconv.FormatInt(after.Unix(), 10),
			},
		},
	})

	if len(connectionIDs) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"source_id": connectionIDs},
		})
	}
	if len(connectors) > 0 {
		connectorsStr := make([]string, 0, len(connectors))
		for _, connector := range connectors {
			connectorsStr = append(connectorsStr, connector.String())
		}
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"source_type": connectorsStr},
		})
	}

	res["size"] = size
	res["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}
	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	query := string(b)
	fmt.Println("query=", query, "index=", summarizer.CostSummeryIndex)
	var response FetchCostHistoryByServicesQueryResponse
	err = client.Search(context.Background(), summarizer.CostSummeryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	for _, hit := range response.Hits.Hits {
		if v, ok := hits[hit.Source.ServiceName]; !ok {
			hits[hit.Source.ServiceName] = []summarizer.ServiceCostSummary{
				hit.Source,
			}
		} else {
			hits[hit.Source.ServiceName] = append(v, hit.Source)
		}
	}

	return hits, nil
}

type FetchDailyCostHistoryByServicesAtTimeResponse struct {
	Aggregations struct {
		SummarizeJobIDGroup struct {
			Buckets []struct {
				ServiceNameGroup struct {
					Buckets []struct {
						Key  string `json:"key"`
						Hits struct {
							Hits struct {
								Hits []struct {
									Source summarizer.ServiceCostSummary `json:"_source"`
								} `json:"hits"`
							} `json:"hits"`
						} `json:"hits"`
					} `json:"buckets"`
				} `json:"service_name_group"`
			} `json:"buckets"`
		} `json:"summarize_job_id_group"`
	} `json:"aggregations"`
}

func FetchDailyCostHistoryByServicesAtTime(client keibi.Client, connectionIDs []string, connectors []source.Type, services []string, at time.Time, size int) (map[string][]summarizer.ServiceCostSummary, error) {
	var filters []any
	filters = append(filters, map[string]any{
		"terms": map[string][]string{"report_type": {string(summarizer.CostProviderSummaryDaily)}},
	})
	if len(services) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"service_name": services},
		})
	}
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"period_end": map[string]string{
				"lte": strconv.FormatInt(at.Unix(), 10),
			},
		},
	})
	if len(connectionIDs) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"source_id": connectionIDs},
		})
	}
	if len(connectors) > 0 {
		connectorsStr := make([]string, 0, len(connectors))
		for _, connector := range connectors {
			connectorsStr = append(connectorsStr, connector.String())
		}
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"source_type": connectorsStr},
		})
	}

	res := make(map[string]any)
	res["size"] = 0
	res["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}
	res["aggs"] = map[string]any{
		"summarize_job_id_group": map[string]any{
			"terms": map[string]any{
				"field": "summarize_job_id",
				"size":  1,
				"order": map[string]string{
					"_term": "desc",
				},
			},
			"aggs": map[string]any{
				"service_name_group": map[string]any{
					"terms": map[string]any{
						"field": "service_name",
						"size":  size,
					},
					"aggs": map[string]any{
						"hits": map[string]any{
							"top_hits": map[string]any{
								"size": size,
								"sort": map[string]string{
									"period_end": "desc",
								},
							},
						},
					},
				},
			},
		},
	}

	query, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	fmt.Printf("query=%s index=%s\n", query, summarizer.CostSummeryIndex)
	var response FetchDailyCostHistoryByServicesAtTimeResponse
	err = client.Search(context.Background(), summarizer.CostSummeryIndex, string(query), &response)
	if err != nil {
		return nil, err
	}

	result := make(map[string][]summarizer.ServiceCostSummary)
	for _, bucket := range response.Aggregations.SummarizeJobIDGroup.Buckets {
		for _, serviceBucket := range bucket.ServiceNameGroup.Buckets {
			for _, hit := range serviceBucket.Hits.Hits.Hits {
				result[serviceBucket.Key] = append(result[serviceBucket.Key], hit.Source)
			}
		}
	}

	return result, nil
}

type FetchCostHistoryByAccountsQueryResponse struct {
	Hits struct {
		Total keibi.SearchTotal `json:"total"`
		Hits  []struct {
			ID      string                           `json:"_id"`
			Score   float64                          `json:"_score"`
			Index   string                           `json:"_index"`
			Type    string                           `json:"_type"`
			Version int64                            `json:"_version,omitempty"`
			Source  summarizer.ConnectionCostSummary `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

func FetchDailyCostHistoryByAccountsBetween(client keibi.Client, connectors []source.Type, connectionIDs []string, before time.Time, after time.Time, size int) (map[string][]summarizer.ConnectionCostSummary, error) {
	before = before.Truncate(24 * time.Hour)
	after = after.Truncate(24 * time.Hour)

	hits := make(map[string][]summarizer.ConnectionCostSummary)
	res := make(map[string]any)
	var filters []any

	filters = append(filters, map[string]any{
		"terms": map[string][]string{"report_type": {string(summarizer.CostConnectionSummaryDaily)}},
	})
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"period_end": map[string]string{
				"lte": strconv.FormatInt(before.Unix(), 10),
			},
		},
	})
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"period_start": map[string]string{
				"gte": strconv.FormatInt(after.Unix(), 10),
			},
		},
	})

	if len(connectionIDs) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"source_id": connectionIDs},
		})
	}
	if len(connectors) > 0 {
		connectorsStr := make([]string, 0, len(connectors))
		for _, connector := range connectors {
			connectorsStr = append(connectorsStr, connector.String())
		}
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"source_type": connectorsStr},
		})
	}

	res["size"] = size
	res["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}
	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	query := string(b)
	fmt.Println("query=", query, "index=", summarizer.CostSummeryIndex)
	var response FetchCostHistoryByAccountsQueryResponse
	err = client.Search(context.Background(), summarizer.CostSummeryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	for _, hit := range response.Hits.Hits {
		if v, ok := hits[hit.Source.SourceID]; !ok {
			hits[hit.Source.SourceID] = []summarizer.ConnectionCostSummary{
				hit.Source,
			}
		} else {
			hits[hit.Source.SourceID] = append(v, hit.Source)
		}
	}

	for _, hitArr := range hits {
		for _, hit := range hitArr {
			switch strings.ToLower(hit.ResourceType) {
			case "aws::costexplorer::byaccountdaily":
				hitCostStr, err := json.Marshal(hit.Cost)
				if err != nil {
					return nil, err
				}
				var hitCost model.CostExplorerByServiceDailyDescription
				err = json.Unmarshal(hitCostStr, &hitCost)
				if err != nil {
					return nil, err
				}
				hit.Cost = hitCost
			}
		}
	}

	return hits, nil
}
