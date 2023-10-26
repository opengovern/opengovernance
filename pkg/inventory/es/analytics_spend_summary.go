package es

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/analytics/es/spend"
	inventoryApi "github.com/kaytu-io/kaytu-engine/pkg/inventory/api"
	"github.com/kaytu-io/kaytu-engine/pkg/summarizer/es"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"strconv"
	"time"
)

type ConnectionDailySpendHistoryByMetric struct {
	Connector     source.Type
	MetricID      string
	MetricName    string
	TotalCost     float64
	StartDateCost float64
	EndDateCost   float64
}

type FetchConnectionDailySpendHistoryByMetricQueryResponse struct {
	Aggregations struct {
		MetricIDGroup struct {
			Buckets []struct {
				Key       string `json:"key"`
				HitSelect struct {
					Hits struct {
						Hits []struct {
							Source spend.ConnectionMetricTrendSummary `json:"_source"`
						} `json:"hits"`
					} `json:"hits"`
				} `json:"hit_select"`
			} `json:"buckets"`
		} `json:"metric_id_group"`
	} `json:"aggregations"`
}

func FetchConnectionDailySpendHistoryByMetric(client kaytu.Client, connectionIDs []string, connectors []source.Type, metricIDs []string, startTime time.Time, endTime time.Time, size int) ([]ConnectionDailySpendHistoryByMetric, error) {
	res := make(map[string]any)
	var filters []any

	if len(metricIDs) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"metric_id": metricIDs},
		})
	}

	filters = append(filters, map[string]any{
		"range": map[string]any{
			"date_epoch": map[string]string{
				"gte": strconv.FormatInt(startTime.UnixMilli(), 10),
				"lte": strconv.FormatInt(endTime.UnixMilli(), 10),
			},
		},
	})

	res["size"] = 0
	res["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}
	res["aggs"] = map[string]any{
		"metric_id_group": map[string]any{
			"terms": map[string]any{
				"field": "metric_id",
				"size":  size,
			},
			"aggs": map[string]any{
				"hit_select": map[string]any{
					"top_hits": map[string]any{
						"size": size,
					},
				},
			},
		},
	}

	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	includeConnectionMap := make(map[string]bool)
	for _, connectionID := range connectionIDs {
		includeConnectionMap[connectionID] = true
	}

	includeConnectorMap := make(map[source.Type]bool)
	for _, connector := range connectors {
		includeConnectorMap[connector] = true
	}

	query := string(b)
	fmt.Println("FetchConnectionDailySpendHistoryByMetric =", query)
	var response FetchConnectionDailySpendHistoryByMetricQueryResponse
	err = client.Search(context.Background(), spend.AnalyticsSpendConnectionSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	hits := make([]ConnectionDailySpendHistoryByMetric, 0, len(response.Aggregations.MetricIDGroup.Buckets))
	for _, metricBucket := range response.Aggregations.MetricIDGroup.Buckets {
		hit := ConnectionDailySpendHistoryByMetric{
			Connector: source.Nil,
		}
		for _, v := range metricBucket.HitSelect.Hits.Hits {
			if hit.MetricID == "" {
				hit.MetricID = v.Source.MetricID
			}
			if hit.MetricName == "" {
				hit.MetricName = v.Source.MetricName
			}
			for _, connectionResult := range v.Source.Connections {
				if (len(connectionIDs) > 0 && !includeConnectionMap[connectionResult.ConnectionID]) ||
					(len(connectors) > 0 && !includeConnectorMap[connectionResult.Connector]) {
					continue
				}
				if hit.Connector == source.Nil {
					hit.Connector = connectionResult.Connector
				}
				hit.TotalCost += connectionResult.CostValue
				if v.Source.Date == startTime.Format("2006-01-02") {
					hit.StartDateCost += connectionResult.CostValue
				}
				if v.Source.Date == endTime.Format("2006-01-02") {
					hit.EndDateCost += connectionResult.CostValue
				}
			}
		}
		hits = append(hits, hit)
	}

	return hits, nil
}

type ConnectionDailySpendHistory struct {
	ConnectionID  string
	TotalCost     float64
	StartDateCost float64
	EndDateCost   float64
}

type FetchConnectionDailySpendHistoryQueryResponse struct {
	Hits struct {
		Hits []struct {
			Source spend.ConnectionMetricTrendSummary `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

func FetchConnectionDailySpendHistory(client kaytu.Client, connectionIDs []string, connectors []source.Type, metricIDs []string, startTime time.Time, endTime time.Time, size int) ([]ConnectionDailySpendHistory, error) {
	res := make(map[string]any)
	var filters []any

	if len(metricIDs) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"metric_id": metricIDs},
		})
	}

	filters = append(filters, map[string]any{
		"range": map[string]any{
			"date_epoch": map[string]string{
				"gte": strconv.FormatInt(startTime.UnixMilli(), 10),
				"lte": strconv.FormatInt(endTime.UnixMilli(), 10),
			},
		},
	})

	res["size"] = size
	res["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}

	includeConnectionMap := make(map[string]bool)
	for _, connectionID := range connectionIDs {
		includeConnectionMap[connectionID] = true
	}
	includeConnectorMap := make(map[source.Type]bool)
	for _, connector := range connectors {
		includeConnectorMap[connector] = true
	}

	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	query := string(b)
	fmt.Println("FetchConnectionDailySpendHistory =", query)
	var response FetchConnectionDailySpendHistoryQueryResponse
	err = client.Search(context.TODO(), spend.AnalyticsSpendConnectionSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	hitsMap := make(map[string]ConnectionDailySpendHistory)
	for _, v := range response.Hits.Hits {
		for _, connectionResult := range v.Source.Connections {
			if (len(connectionIDs) > 0 && !includeConnectionMap[connectionResult.ConnectionID]) ||
				(len(connectors) > 0 && !includeConnectorMap[connectionResult.Connector]) {
				continue
			}
			connHit, ok := hitsMap[connectionResult.ConnectionID]
			if !ok {
				connHit = ConnectionDailySpendHistory{
					ConnectionID: connectionResult.ConnectionID,
				}
			}
			connHit.TotalCost += connectionResult.CostValue
			if v.Source.Date == startTime.Format("2006-01-02") {
				connHit.StartDateCost += connectionResult.CostValue
			}
			if v.Source.Date == endTime.Format("2006-01-02") {
				connHit.EndDateCost += connectionResult.CostValue
			}
			hitsMap[connectionResult.ConnectionID] = connHit
		}
	}

	hits := make([]ConnectionDailySpendHistory, 0, len(hitsMap))
	for _, v := range hitsMap {
		hits = append(hits, v)
	}
	return hits, nil
}

type ConnectorDailySpendHistoryByMetric struct {
	Connector     string
	MetricID      string
	MetricName    string
	TotalCost     float64
	StartDateCost float64
	EndDateCost   float64
}

type FetchConnectorDailySpendHistoryByMetricQueryResponse struct {
	Aggregations struct {
		MetricIDGroup struct {
			Buckets []struct {
				Key       string `json:"key"`
				HitSelect struct {
					Hits struct {
						Hits []struct {
							Source spend.ConnectorMetricTrendSummary `json:"_source"`
						} `json:"hits"`
					} `json:"hits"`
				} `json:"hit_select"`
			} `json:"buckets"`
		} `json:"metric_id_group"`
	} `json:"aggregations"`
}

func FetchConnectorDailySpendHistoryByMetric(client kaytu.Client, connectors []source.Type, metricIDs []string, startTime time.Time, endTime time.Time, size int) ([]ConnectorDailySpendHistoryByMetric, error) {
	res := make(map[string]any)
	var filters []any

	if len(metricIDs) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"metric_id": metricIDs},
		})
	}

	filters = append(filters, map[string]any{
		"range": map[string]any{
			"date_epoch": map[string]string{
				"gte": strconv.FormatInt(startTime.UnixMilli(), 10),
				"lte": strconv.FormatInt(endTime.UnixMilli(), 10),
			},
		},
	})

	res["size"] = 0
	res["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}
	res["aggs"] = map[string]any{
		"metric_id_group": map[string]any{
			"terms": map[string]any{
				"field": "metric_id",
				"size":  size,
			},
			"aggs": map[string]any{
				"hit_select": map[string]any{
					"top_hits": map[string]any{
						"size": size,
					},
				},
			},
		},
	}

	includeConnectorMap := make(map[string]bool)
	for _, connector := range connectors {
		includeConnectorMap[connector.String()] = true
	}

	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	query := string(b)
	fmt.Println("FetchConnectorDailySpendHistoryByMetric =", query)
	var response FetchConnectorDailySpendHistoryByMetricQueryResponse
	err = client.Search(context.Background(), spend.AnalyticsSpendConnectionSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	var hits []ConnectorDailySpendHistoryByMetric
	for _, metricBucket := range response.Aggregations.MetricIDGroup.Buckets {
		hit := ConnectorDailySpendHistoryByMetric{
			Connector:     source.Nil,
			MetricID:      "",
			MetricName:    "",
			TotalCost:     0,
			StartDateCost: 0,
			EndDateCost:   0,
		}
		for _, v := range metricBucket.HitSelect.Hits.Hits {
			if hit.MetricID == "" {
				hit.MetricID = v.Source.MetricID
			}
			if hit.MetricName == "" {
				hit.MetricName = v.Source.MetricName
			}
			for _, connectorResult := range v.Source.Connectors {
				if len(connectors) > 0 && !includeConnectorMap[connectorResult.Connector.String()] {
					continue
				}
				if hit.Connector == source.Nil {
					hit.Connector = connectorResult.Connector.String()
				}
				hit.TotalCost += connectorResult.CostValue
				if v.Source.Date == startTime.Format("2006-01-02") {
					hit.StartDateCost += connectorResult.CostValue
				}
				if v.Source.Date == endTime.Format("2006-01-02") {
					hit.EndDateCost += connectorResult.CostValue
				}
			}
		}
		hits = append(hits, hit)
	}

	return hits, nil
}

type ConnectionSpendTrendQueryResponse struct {
	Aggregations struct {
		DateGroup struct {
			Buckets []struct {
				Key       string `json:"key"`
				HitSelect struct {
					Hits struct {
						Hits []struct {
							Source spend.ConnectionMetricTrendSummary `json:"_source"`
						} `json:"hits"`
					} `json:"hits"`
				} `json:"hit_select"`
			} `json:"buckets"`
		} `json:"date_group"`
	} `json:"aggregations"`
}

func FetchConnectionSpendTrend(client kaytu.Client, granularity inventoryApi.TableGranularityType, metricIds []string, connectionIDs []string, connectors []source.Type, startTime, endTime time.Time) (map[string]DatapointWithFailures, error) {
	query := make(map[string]any)
	var filters []any

	if len(metricIds) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"metric_id": metricIds},
		})
	}

	filters = append(filters, map[string]any{
		"range": map[string]any{
			"date_epoch": map[string]string{
				"gte": strconv.FormatInt(startTime.UnixMilli(), 10),
				"lte": strconv.FormatInt(endTime.UnixMilli(), 10),
			},
		},
	})

	granularityField := "date"
	if granularity == inventoryApi.TableGranularityTypeMonthly {
		granularityField = "month"
	} else if granularity == inventoryApi.TableGranularityTypeYearly {
		granularityField = "year"
	}

	query["size"] = 0
	query["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}
	query["aggs"] = map[string]any{
		"date_group": map[string]any{
			"terms": map[string]any{
				"field": granularityField,
				"size":  es.EsFetchPageSize,
			},
			"aggs": map[string]any{
				"hit_select": map[string]any{
					"top_hits": map[string]any{
						"size": es.EsFetchPageSize,
					},
				},
			},
		},
	}

	includeConnectionMap := make(map[string]bool)
	for _, connectionID := range connectionIDs {
		includeConnectionMap[connectionID] = true
	}

	includeConnectorMap := make(map[source.Type]bool)
	for _, connector := range connectors {
		includeConnectorMap[connector] = true
	}

	queryJson, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}
	fmt.Printf("FetchConnectionSpendTrend = %s\n", queryJson)

	var response ConnectionSpendTrendQueryResponse
	err = client.Search(context.TODO(), spend.AnalyticsSpendConnectionSummaryIndex, string(queryJson), &response)
	if err != nil {
		return nil, err
	}

	result := make(map[string]DatapointWithFailures)
	for _, bucket := range response.Aggregations.DateGroup.Buckets {
		res := DatapointWithFailures{}
		for _, hit := range bucket.HitSelect.Hits.Hits {
			for _, connection := range hit.Source.Connections {
				if (len(connectionIDs) > 0 && !includeConnectionMap[connection.ConnectionID]) ||
					(len(connectors) > 0 && !includeConnectorMap[connection.Connector]) {
					continue
				}
				res.TotalConnections++
				if connection.IsJobSuccessful {
					res.TotalSuccessfulConnections++
				}
				res.Cost += connection.CostValue
			}
		}
		result[bucket.Key] = res
	}

	return result, nil
}

type ConnectorSpendTrendQueryResponse struct {
	Aggregations struct {
		DateGroup struct {
			Buckets []struct {
				Key       string `json:"key"`
				HitSelect struct {
					Hits struct {
						Hits []struct {
							Source spend.ConnectorMetricTrendSummary `json:"_source"`
						} `json:"hits"`
					} `json:"hits"`
				} `json:"hit_select"`
			} `json:"buckets"`
		} `json:"date_group"`
	} `json:"aggregations"`
}

func FetchConnectorSpendTrend(client kaytu.Client, granularity inventoryApi.TableGranularityType, metricIds []string, connectors []source.Type, startTime, endTime time.Time) (map[string]DatapointWithFailures, error) {
	query := make(map[string]any)
	var filters []any

	if len(metricIds) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"metric_id": metricIds},
		})
	}
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"date_epoch": map[string]string{
				"gte": strconv.FormatInt(startTime.UnixMilli(), 10),
				"lte": strconv.FormatInt(endTime.UnixMilli(), 10),
			},
		},
	})

	granularityField := "date"
	if granularity == inventoryApi.TableGranularityTypeMonthly {
		granularityField = "month"
	} else if granularity == inventoryApi.TableGranularityTypeYearly {
		granularityField = "year"
	}

	query["size"] = 0
	query["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}
	query["aggs"] = map[string]any{
		"date_group": map[string]any{
			"terms": map[string]any{
				"field": granularityField,
				"size":  es.EsFetchPageSize,
			},
			"aggs": map[string]any{
				"hit_select": map[string]any{
					"top_hits": map[string]any{
						"size": es.EsFetchPageSize,
					},
				},
			},
		},
	}

	includeConnectorMap := make(map[source.Type]bool)
	for _, connector := range connectors {
		includeConnectorMap[connector] = true
	}

	queryJson, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}
	fmt.Printf("FetchConnectorSpendTrend = %s\n", queryJson)

	var response ConnectorSpendTrendQueryResponse
	err = client.Search(context.Background(), spend.AnalyticsSpendConnectorSummaryIndex, string(queryJson), &response)
	if err != nil {
		return nil, err
	}

	result := make(map[string]DatapointWithFailures)
	for _, bucket := range response.Aggregations.DateGroup.Buckets {
		res := DatapointWithFailures{}
		perConnectorTotalConnections := make(map[source.Type]int64)
		perConnectorTotalSuccessfulConnections := make(map[source.Type]int64)
		for _, hit := range bucket.HitSelect.Hits.Hits {
			for _, connector := range hit.Source.Connectors {
				if len(connectors) > 0 && !includeConnectorMap[connector.Connector] {
					continue
				}
				perConnectorTotalConnections[connector.Connector] = max(perConnectorTotalConnections[connector.Connector], connector.TotalConnections)
				perConnectorTotalSuccessfulConnections[connector.Connector] = max(perConnectorTotalSuccessfulConnections[connector.Connector], connector.TotalSuccessfulConnections)
				res.Cost += connector.CostValue
			}
		}
		for connector, totalConnections := range perConnectorTotalConnections {
			res.TotalConnections += totalConnections
			res.TotalSuccessfulConnections += perConnectorTotalSuccessfulConnections[connector]
		}
		result[bucket.Key] = res
	}

	return result, nil
}

type FetchSpendByMetricQueryResponse struct {
	Aggregations struct {
		MetricIDGroup struct {
			Buckets []struct {
				Key       string `json:"key"`
				HitSelect struct {
					Hits struct {
						Hits []struct {
							Source spend.ConnectionMetricTrendSummary `json:"_source"`
						} `json:"hits"`
					} `json:"hits"`
				} `json:"hit_select"`
			} `json:"buckets"`
		} `json:"metric_id_group"`
	} `json:"aggregations"`
}

type SpendMetricResp struct {
	MetricName string
	CostValue  float64
}

func FetchSpendByMetric(client kaytu.Client, connectionIDs []string, connectors []source.Type, metricIDs []string, startTime time.Time, endTime time.Time, size int) (map[string]SpendMetricResp, error) {
	res := make(map[string]any)
	var filters []any

	if len(metricIDs) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"metric_id": metricIDs},
		})
	}

	filters = append(filters, map[string]any{
		"range": map[string]any{
			"date_epoch": map[string]string{
				"gte": strconv.FormatInt(startTime.UnixMilli(), 10),
				"lte": strconv.FormatInt(endTime.UnixMilli(), 10),
			},
		},
	})

	res["size"] = 0
	res["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}
	res["aggs"] = map[string]any{
		"metric_id_group": map[string]any{
			"terms": map[string]any{
				"field": "metric_id",
				"size":  size,
			},
			"aggs": map[string]any{
				"hit_select": map[string]any{
					"top_hits": map[string]any{
						"size": size,
					},
				},
			},
		},
	}

	includeConnectionMap := make(map[string]bool)
	for _, connectionID := range connectionIDs {
		includeConnectionMap[connectionID] = true
	}
	includeConnectorMap := make(map[source.Type]bool)
	for _, connector := range connectors {
		includeConnectorMap[connector] = true
	}

	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	query := string(b)
	fmt.Println("FetchSpendByMetric =", query)
	var response FetchSpendByMetricQueryResponse
	err = client.Search(context.Background(), spend.AnalyticsSpendConnectionSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	resp := map[string]SpendMetricResp{}
	for _, metricBucket := range response.Aggregations.MetricIDGroup.Buckets {
		for _, v := range metricBucket.HitSelect.Hits.Hits {
			if len(connectionIDs) == 0 && len(connectors) == 0 {
				resp[metricBucket.Key] = SpendMetricResp{
					MetricName: v.Source.MetricName,
					CostValue:  v.Source.TotalCostValue,
				}
				continue
			}
			for _, connectionResult := range v.Source.Connections {
				if (len(connectionIDs) > 0 && !includeConnectionMap[connectionResult.ConnectionID]) ||
					(len(connectors) > 0 && !includeConnectorMap[connectionResult.Connector]) {
					continue
				}
				metricResp, ok := resp[metricBucket.Key]
				if !ok {
					metricResp = SpendMetricResp{
						MetricName: v.Source.MetricName,
					}
				}
				metricResp.CostValue += connectionResult.CostValue
				resp[metricBucket.Key] = metricResp
			}
		}
	}
	return resp, nil
}

type DimensionTrend struct {
	DimensionID   string
	Connector     source.Type
	DimensionName string
	Trend         map[string]float64
}

type SpendTableByDimensionQueryResponse struct {
	Aggregations struct {
		DateGroup struct {
			Buckets []struct {
				Key       string `json:"key"`
				HitSelect struct {
					Hits struct {
						Hits []struct {
							Source spend.ConnectionMetricTrendSummary `json:"_source"`
						} `json:"hits"`
					} `json:"hits"`
				} `json:"hit_select"`
			} `json:"buckets"`
		} `json:"date_group"`
	} `json:"aggregations"`
}

func FetchSpendTableByDimension(client kaytu.Client, dimension inventoryApi.DimensionType, connectionIds []string, connectors []source.Type, metricIds []string, startTime, endTime time.Time) ([]DimensionTrend, error) {
	query := make(map[string]any)
	var filters []any

	filters = append(filters, map[string]any{
		"range": map[string]any{
			"date_epoch": map[string]string{
				"gte": strconv.FormatInt(startTime.UnixMilli(), 10),
				"lte": strconv.FormatInt(endTime.UnixMilli(), 10),
			},
		},
	})

	if len(metricIds) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"metric_id": metricIds},
		})
	}

	query["size"] = 0
	query["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}
	query["aggs"] = map[string]any{
		"metric_group": map[string]any{
			"date_group": map[string]any{
				"terms": map[string]any{
					"field": "date",
					"size":  es.EsFetchPageSize,
				},
				"aggs": map[string]any{
					"hit_select": map[string]any{
						"top_hits": map[string]any{
							"size": es.EsFetchPageSize,
						},
					},
				},
			},
		},
	}

	includeConnectionMap := make(map[string]bool)
	for _, connectionID := range connectionIds {
		includeConnectionMap[connectionID] = true
	}

	includeConnectorMap := make(map[source.Type]bool)
	for _, connector := range connectors {
		includeConnectorMap[connector] = true
	}

	queryJson, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}
	fmt.Printf("FetchSpendTableByDimension = %s\n", queryJson)

	var response SpendTableByDimensionQueryResponse
	err = client.Search(context.Background(), spend.AnalyticsSpendConnectionSummaryIndex, string(queryJson), &response)
	if err != nil {
		return nil, err
	}

	var result map[string]DimensionTrend
	fmt.Println(response)

	for _, dateBucket := range response.Aggregations.DateGroup.Buckets {
		for _, hit := range dateBucket.HitSelect.Hits.Hits {
			for _, connectionResult := range hit.Source.Connections {
				if (len(connectionIds) > 0 && !includeConnectionMap[connectionResult.ConnectionID]) ||
					(len(connectors) > 0 && !includeConnectorMap[connectionResult.Connector]) {
					continue
				}
				key := ""
				switch dimension {
				case inventoryApi.DimensionTypeConnection:
					key = connectionResult.ConnectionID
				case inventoryApi.DimensionTypeMetric:
					key = hit.Source.MetricID
				}
				mt, ok := result[key]
				if !ok {
					mt = DimensionTrend{
						DimensionID: key,
						Connector:   connectionResult.Connector,
						Trend:       make(map[string]float64),
					}
					switch dimension {
					case inventoryApi.DimensionTypeConnection:
						mt.DimensionName = connectionResult.ConnectionName
					case inventoryApi.DimensionTypeMetric:
						mt.DimensionName = hit.Source.MetricName
					default:
						return nil, errors.New("dimension is not supported")
					}
				}
				mt.Trend[dateBucket.Key] += connectionResult.CostValue
				result[key] = mt
			}
		}
	}

	resultList := make([]DimensionTrend, 0, len(result))
	for _, v := range result {
		resultList = append(resultList, v)
	}

	return resultList, nil
}
