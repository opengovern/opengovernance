package es

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/kaytu-io/kaytu-aws-describer/aws/model"
	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	summarizer "gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"
)

type FetchCostHistoryByServicesQueryResponse struct {
	Aggregations struct {
		ConnectorGroup struct {
			Buckets []struct {
				Key              string `json:"key"`
				ServiceNameGroup struct {
					Buckets []struct {
						Key               string `json:"key"`
						CostValueSumGroup struct {
							Value float64 `json:"value"`
						} `json:"cost_value_sum_group"`
					} `json:"buckets"`
				} `json:"service_name_group"`
			}
		}
	} `json:"aggregations"`
}

func FetchDailyCostHistoryByServicesBetween(client keibi.Client, connectionIDs []string, connectors []source.Type, services []string, startTime time.Time, endTime time.Time, size int) (map[string]map[string]float64, error) {
	endTime = endTime.Truncate(24 * time.Hour)
	startTime = startTime.Truncate(24 * time.Hour)

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
				"lte": strconv.FormatInt(endTime.Unix(), 10),
			},
		},
	})
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"period_start": map[string]string{
				"gte": strconv.FormatInt(startTime.Unix(), 10),
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

	res["size"] = 0
	res["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}
	res["aggs"] = map[string]any{
		"connector_group": map[string]any{
			"terms": map[string]any{
				"field": "source_type",
			},
			"aggs": map[string]any{
				"service_name_group": map[string]any{
					"terms": map[string]any{
						"field": "service_name",
					},
					"aggs": map[string]any{
						"cost_value_sum_group": map[string]any{
							"sum": map[string]any{
								"field": "cost_value",
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
	var response FetchCostHistoryByServicesQueryResponse
	err = client.Search(context.Background(), summarizer.CostSummeryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	hits := make(map[string]map[string]float64)
	for _, connectorBucket := range response.Aggregations.ConnectorGroup.Buckets {
		if _, ok := hits[connectorBucket.Key]; !ok {
			hits[connectorBucket.Key] = make(map[string]float64)
		}
		for _, serviceNameBucket := range connectorBucket.ServiceNameGroup.Buckets {
			hits[connectorBucket.Key][serviceNameBucket.Key] += serviceNameBucket.CostValueSumGroup.Value
		}
	}

	return hits, nil
}

type FetchDailyCostHistoryByServicesAtTimeResponse struct {
	Aggregations struct {
		ServiceNameGroup struct {
			Buckets []struct {
				Key               string `json:"key"`
				ConnectionIDGroup struct {
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
				} `json:"connection_id_group"`
			} `json:"buckets"`
		} `json:"service_name_group"`
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
		"service_name_group": map[string]any{
			"terms": map[string]any{
				"field": "service_name",
				"size":  size,
			},
			"aggs": map[string]any{
				"connection_id_group": map[string]any{
					"terms": map[string]any{
						"field": "source_id",
						"size":  size,
					},
					"aggs": map[string]any{
						"hits": map[string]any{
							"top_hits": map[string]any{
								"size": 1,
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
	for _, bucket := range response.Aggregations.ServiceNameGroup.Buckets {
		for _, connectionBucket := range bucket.ConnectionIDGroup.Buckets {
			for _, hit := range connectionBucket.Hits.Hits.Hits {
				result[bucket.Key] = append(result[bucket.Key], hit.Source)
			}
		}
	}

	return result, nil
}

type FetchDailyCostTrendByServicesBetweenResponse struct {
	Aggregations struct {
		ServiceNameGroup struct {
			Buckets []struct {
				Key                 string `json:"key"`
				PeriodEndRangeGroup struct {
					Buckets []struct {
						From         float64 `json:"from"`
						To           float64 `json:"to"`
						CostSumGroup struct {
							Value float64 `json:"value"`
						} `json:"cost_sum_group"`
					} `json:"buckets"`
				} `json:"period_end_range_group"`
			} `json:"buckets"`
		} `json:"service_name_group"`
	} `json:"aggregations"`
}

func FetchDailyCostTrendByServicesBetween(client keibi.Client, connectionIDs []string, connectors []source.Type, services []string, startTime, endTime time.Time, datapointCount int) (map[string]map[int]float64, error) {
	startTime = startTime.Truncate(time.Hour * 24)
	endTime = endTime.Truncate(time.Hour * 24)

	query := make(map[string]any)
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
				"lte": strconv.FormatInt(endTime.Unix(), 10),
			},
		},
	})
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"period_start": map[string]string{
				"gte": strconv.FormatInt(startTime.Unix(), 10),
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

	startTimeUnix := startTime.Unix()
	endTimeUnix := endTime.Unix()
	step := int(math.Ceil(float64(endTimeUnix-startTimeUnix) / float64(datapointCount)))
	ranges := make([]map[string]any, 0, datapointCount)
	for i := 0; i < datapointCount; i++ {
		ranges = append(ranges, map[string]any{
			"from": startTimeUnix + int64(i*step),
			"to":   startTimeUnix + int64((i+1)*step),
		})
	}

	query["size"] = 0
	query["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}
	query["aggs"] = map[string]any{
		"service_name_group": map[string]any{
			"terms": map[string]any{
				"field": "service_name",
				"size":  10000,
			},
			"aggs": map[string]any{
				"period_end_range_group": map[string]any{
					"range": map[string]any{
						"field":  "period_end",
						"ranges": ranges,
					},
					"aggs": map[string]any{
						"cost_sum_group": map[string]any{
							"sum": map[string]string{
								"field": "cost_value",
							},
						},
					},
				},
			},
		},
	}

	queryJson, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}
	fmt.Printf("query=%s index=%s\n", queryJson, summarizer.CostSummeryIndex)

	var response FetchDailyCostTrendByServicesBetweenResponse
	err = client.Search(context.Background(), summarizer.CostSummeryIndex, string(queryJson), &response)
	if err != nil {
		return nil, err
	}

	result := make(map[string]map[int]float64)
	for _, serviceNameBucket := range response.Aggregations.ServiceNameGroup.Buckets {
		if _, ok := result[serviceNameBucket.Key]; !ok {
			result[serviceNameBucket.Key] = make(map[int]float64)
		}
		for _, periodEndRangeBucket := range serviceNameBucket.PeriodEndRangeGroup.Buckets {
			rangeBucketKey := int((periodEndRangeBucket.From + periodEndRangeBucket.To) / 2)
			result[serviceNameBucket.Key][rangeBucketKey] = periodEndRangeBucket.CostSumGroup.Value
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
