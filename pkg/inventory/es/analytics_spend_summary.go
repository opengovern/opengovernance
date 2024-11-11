package es

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/opengovern/og-util/pkg/integration"
	"strconv"
	"time"

	"github.com/opengovern/og-util/pkg/opengovernance-es-sdk"
	"github.com/opengovern/opengovernance/pkg/analytics/es/spend"
	inventoryAPI "github.com/opengovern/opengovernance/pkg/inventory/api"
	"go.uber.org/zap"
)

const EsFetchPageSize = 10000

type ConnectionDailySpendHistoryByMetric struct {
	IntegrationType integration.Type
	MetricID        string
	MetricName      string
	TotalCost       float64
	StartDateCost   float64
	EndDateCost     float64
}

type FetchConnectionDailySpendHistoryByMetricQueryResponse struct {
	Aggregations struct {
		MetricIDGroup struct {
			Buckets []struct {
				Key       string `json:"key"`
				HitSelect struct {
					Hits struct {
						Hits []struct {
							Source spend.IntegrationMetricTrendSummary `json:"_source"`
						} `json:"hits"`
					} `json:"hits"`
				} `json:"hit_select"`
			} `json:"buckets"`
		} `json:"metric_id_group"`
	} `json:"aggregations"`
}

func FetchConnectionDailySpendHistoryByMetric(ctx context.Context, client opengovernance.Client, connectionIDs []string, integrationTypes []integration.Type, metricIDs []string, startTime time.Time, endTime time.Time, size int) ([]ConnectionDailySpendHistoryByMetric, error) {
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

	includeIntegrationTypeMap := make(map[integration.Type]bool)
	for _, integrationType := range integrationTypes {
		includeIntegrationTypeMap[integrationType] = true
	}

	query := string(b)
	fmt.Println("FetchConnectionDailySpendHistoryByMetric =", query)
	var response FetchConnectionDailySpendHistoryByMetricQueryResponse
	err = client.Search(ctx, spend.AnalyticsSpendIntegrationSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	hits := make([]ConnectionDailySpendHistoryByMetric, 0, len(response.Aggregations.MetricIDGroup.Buckets))
	for _, metricBucket := range response.Aggregations.MetricIDGroup.Buckets {
		hit := ConnectionDailySpendHistoryByMetric{
			IntegrationType: "",
		}
		for _, v := range metricBucket.HitSelect.Hits.Hits {
			if hit.MetricID == "" {
				hit.MetricID = v.Source.MetricID
			}
			if hit.MetricName == "" {
				hit.MetricName = v.Source.MetricName
			}
			for _, connectionResult := range v.Source.Integrations {
				if (len(connectionIDs) > 0 && !includeConnectionMap[connectionResult.IntegrationID]) ||
					(len(integrationTypes) > 0 && !includeIntegrationTypeMap[connectionResult.IntegrationType]) {
					continue
				}
				if hit.IntegrationType == "" {
					hit.IntegrationType = connectionResult.IntegrationType
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
			Source spend.IntegrationMetricTrendSummary `json:"_source"`
			Sort   []any                               `json:"sort"`
		} `json:"hits"`
	} `json:"hits"`
}

func FetchConnectionDailySpendHistory(ctx context.Context, client opengovernance.Client, connectionIDs []string, connectors []integration.Type, metricIDs []string, startTime time.Time, endTime time.Time, size int) ([]ConnectionDailySpendHistory, error) {
	filterPaths := make([]string, 0)
	filterPaths = append(filterPaths, "hits.hits.sort")
	filterPaths = append(filterPaths, "hits.hits._source.connections.connection_id")
	filterPaths = append(filterPaths, "hits.hits._source.connections.connector")
	filterPaths = append(filterPaths, "hits.hits._source.connections.cost_value")

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
	includeConnectorMap := make(map[integration.Type]bool)
	for _, connector := range connectors {
		includeConnectorMap[connector] = true
	}

	hitsMap := make(map[string]ConnectionDailySpendHistory)
	var searchAfter []any
	res["sort"] = map[string]string{
		"date_epoch": "desc",
		"_id":        "asc",
	}
	for {
		if len(searchAfter) > 0 {
			res["search_after"] = searchAfter
		}
		b, err := json.Marshal(res)
		if err != nil {
			return nil, err
		}

		query := string(b)
		fmt.Println("FetchConnectionDailySpendHistory =", query)
		var response FetchConnectionDailySpendHistoryQueryResponse
		err = client.SearchWithFilterPath(ctx, spend.AnalyticsSpendIntegrationSummaryIndex, query, filterPaths, &response)
		if err != nil {
			return nil, err
		}

		if len(response.Hits.Hits) == 0 {
			break
		}

		for _, v := range response.Hits.Hits {
			for _, connectionResult := range v.Source.Integrations {
				if (len(connectionIDs) > 0 && !includeConnectionMap[connectionResult.IntegrationID]) ||
					(len(connectors) > 0 && !includeConnectorMap[connectionResult.IntegrationType]) {
					continue
				}
				connHit, ok := hitsMap[connectionResult.IntegrationID]
				if !ok {
					connHit = ConnectionDailySpendHistory{
						ConnectionID: connectionResult.IntegrationID,
					}
				}
				connHit.TotalCost += connectionResult.CostValue
				if v.Source.Date == startTime.Format("2006-01-02") {
					connHit.StartDateCost += connectionResult.CostValue
				}
				if v.Source.Date == endTime.Format("2006-01-02") {
					connHit.EndDateCost += connectionResult.CostValue
				}
				hitsMap[connectionResult.IntegrationID] = connHit
			}
			searchAfter = v.Sort
		}
	}
	hits := make([]ConnectionDailySpendHistory, 0, len(hitsMap))
	for _, v := range hitsMap {
		hits = append(hits, v)
	}
	return hits, nil
}

type IntegrationTypeDailySpendHistoryByMetric struct {
	IntegrationType string
	MetricID        string
	MetricName      string
	TotalCost       float64
	StartDateCost   float64
	EndDateCost     float64
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

func FetchConnectorDailySpendHistoryByMetric(ctx context.Context, client opengovernance.Client, integrationTypes []integration.Type, metricIDs []string, startTime time.Time, endTime time.Time, size int) ([]IntegrationTypeDailySpendHistoryByMetric, error) {
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

	includeIntegrationTypeMap := make(map[string]bool)
	for _, integrationType := range integrationTypes {
		includeIntegrationTypeMap[integrationType.String()] = true
	}

	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	query := string(b)
	fmt.Println("FetchConnectorDailySpendHistoryByMetric =", query)
	var response FetchConnectorDailySpendHistoryByMetricQueryResponse
	err = client.Search(ctx, spend.AnalyticsSpendIntegrationTypeSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	var hits []IntegrationTypeDailySpendHistoryByMetric
	for _, metricBucket := range response.Aggregations.MetricIDGroup.Buckets {
		hit := IntegrationTypeDailySpendHistoryByMetric{
			IntegrationType: "",
			MetricID:        "",
			MetricName:      "",
			TotalCost:       0,
			StartDateCost:   0,
			EndDateCost:     0,
		}
		for _, v := range metricBucket.HitSelect.Hits.Hits {
			if hit.MetricID == "" {
				hit.MetricID = v.Source.MetricID
			}
			if hit.MetricName == "" {
				hit.MetricName = v.Source.MetricName
			}
			for _, connectorResult := range v.Source.IntegrationTypes {
				if len(integrationTypes) > 0 && !includeIntegrationTypeMap[connectorResult.Connector.String()] {
					continue
				}
				if hit.IntegrationType == "" {
					hit.IntegrationType = connectorResult.Connector.String()
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
							Source spend.IntegrationMetricTrendSummary `json:"_source"`
						} `json:"hits"`
					} `json:"hits"`
				} `json:"hit_select"`
			} `json:"buckets"`
		} `json:"date_group"`
	} `json:"aggregations"`
}

func FetchConnectionSpendTrend(ctx context.Context, client opengovernance.Client, granularity inventoryAPI.TableGranularityType, metricIds []string, connectionIDs []string, connectors []integration.Type, startTime, endTime time.Time) (map[string]DatapointWithFailures, error) {
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
	if granularity == inventoryAPI.TableGranularityTypeMonthly {
		granularityField = "month"
	} else if granularity == inventoryAPI.TableGranularityTypeYearly {
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
				"size":  EsFetchPageSize,
			},
			"aggs": map[string]any{
				"hit_select": map[string]any{
					"top_hits": map[string]any{
						"size": EsFetchPageSize,
					},
				},
			},
		},
	}

	includeConnectionMap := make(map[string]bool)
	for _, connectionID := range connectionIDs {
		includeConnectionMap[connectionID] = true
	}

	includeConnectorMap := make(map[integration.Type]bool)
	for _, connector := range connectors {
		includeConnectorMap[connector] = true
	}

	queryJson, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}
	fmt.Printf("FetchConnectionSpendTrend = %s\n", queryJson)

	var response ConnectionSpendTrendQueryResponse
	err = client.Search(ctx, spend.AnalyticsSpendIntegrationSummaryIndex, string(queryJson), &response)
	if err != nil {
		return nil, err
	}

	result := make(map[string]DatapointWithFailures)
	for _, bucket := range response.Aggregations.DateGroup.Buckets {
		res := DatapointWithFailures{
			CostStacked: map[string]float64{},
		}
		for _, hit := range bucket.HitSelect.Hits.Hits {
			for _, connection := range hit.Source.Integrations {
				if (len(connectionIDs) > 0 && !includeConnectionMap[connection.IntegrationID]) ||
					(len(connectors) > 0 && !includeConnectorMap[connection.IntegrationType]) {
					continue
				}
				res.TotalConnections++
				if connection.IsJobSuccessful {
					res.TotalSuccessfulConnections++
				}
				res.Cost += connection.CostValue
				res.CostStacked[hit.Source.MetricID] += connection.CostValue
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

func FetchConnectorSpendTrend(ctx context.Context, client opengovernance.Client, granularity inventoryAPI.TableGranularityType, metricIds []string, connectors []integration.Type, startTime, endTime time.Time) (map[string]DatapointWithFailures, error) {
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
	if granularity == inventoryAPI.TableGranularityTypeMonthly {
		granularityField = "month"
	} else if granularity == inventoryAPI.TableGranularityTypeYearly {
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
				"size":  EsFetchPageSize,
			},
			"aggs": map[string]any{
				"hit_select": map[string]any{
					"top_hits": map[string]any{
						"size": EsFetchPageSize,
					},
				},
			},
		},
	}

	includeConnectorMap := make(map[integration.Type]bool)
	for _, connector := range connectors {
		includeConnectorMap[connector] = true
	}

	queryJson, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}
	fmt.Printf("FetchConnectorSpendTrend = %s\n", queryJson)

	var response ConnectorSpendTrendQueryResponse
	err = client.Search(ctx, spend.AnalyticsSpendIntegrationTypeSummaryIndex, string(queryJson), &response)
	if err != nil {
		return nil, err
	}

	result := make(map[string]DatapointWithFailures)
	for _, bucket := range response.Aggregations.DateGroup.Buckets {
		res := DatapointWithFailures{
			CostStacked: map[string]float64{},
		}
		perConnectorTotalConnections := make(map[integration.Type]int64)
		perConnectorTotalSuccessfulConnections := make(map[integration.Type]int64)
		for _, hit := range bucket.HitSelect.Hits.Hits {
			for _, connector := range hit.Source.IntegrationTypes {
				if len(connectors) > 0 && !includeConnectorMap[connector.Connector] {
					continue
				}
				perConnectorTotalConnections[connector.Connector] = max(perConnectorTotalConnections[connector.Connector], connector.TotalConnections)
				perConnectorTotalSuccessfulConnections[connector.Connector] = max(perConnectorTotalSuccessfulConnections[connector.Connector], connector.TotalSuccessfulConnections)
				res.Cost += connector.CostValue
				res.CostStacked[hit.Source.MetricID] += connector.CostValue
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

type FetchSpendByMetricConnectionQueryResponse struct {
	Aggregations struct {
		MetricIDGroup struct {
			Buckets []struct {
				Key       string `json:"key"`
				HitSelect struct {
					Hits struct {
						Hits []struct {
							Source spend.IntegrationMetricTrendSummary `json:"_source"`
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

func FetchSpendByMetric(ctx context.Context, client opengovernance.Client, connectionIDs []string, connectors []integration.Type, metricIDs []string, startTime time.Time, endTime time.Time, size int) (map[string]SpendMetricResp, error) {
	if len(connectionIDs) > 0 {
		return FetchSpendByMetricConnection(ctx, client, connectionIDs, connectors, metricIDs, startTime, endTime, size)
	} else {
		return FetchSpendByMetricConnector(ctx, client, connectors, metricIDs, startTime, endTime, size)
	}
}

func FetchSpendByMetricConnection(ctx context.Context, client opengovernance.Client, connectionIDs []string, connectors []integration.Type, metricIDs []string, startTime time.Time, endTime time.Time, size int) (map[string]SpendMetricResp, error) {
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
	includeConnectorMap := make(map[integration.Type]bool)
	for _, connector := range connectors {
		includeConnectorMap[connector] = true
	}

	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	query := string(b)
	fmt.Println("FetchSpendByMetricConnection =", query)
	var response FetchSpendByMetricConnectionQueryResponse
	err = client.Search(ctx, spend.AnalyticsSpendIntegrationSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	resp := map[string]SpendMetricResp{}
	for _, metricBucket := range response.Aggregations.MetricIDGroup.Buckets {
		for _, v := range metricBucket.HitSelect.Hits.Hits {
			if len(connectionIDs) == 0 && len(connectors) == 0 {
				metricResp := resp[metricBucket.Key]
				metricResp.MetricName = v.Source.MetricName
				metricResp.CostValue += v.Source.TotalCostValue
				resp[metricBucket.Key] = metricResp
				continue
			}
			for _, connectionResult := range v.Source.Integrations {
				if (len(connectionIDs) > 0 && !includeConnectionMap[connectionResult.IntegrationID]) ||
					(len(connectors) > 0 && !includeConnectorMap[connectionResult.IntegrationType]) {
					continue
				}
				metricResp := resp[metricBucket.Key]
				metricResp.MetricName = v.Source.MetricName
				metricResp.CostValue += connectionResult.CostValue
				resp[metricBucket.Key] = metricResp
			}
		}
	}
	return resp, nil
}

type FetchSpendByMetricConnectorQueryResponse struct {
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

func FetchSpendByMetricConnector(ctx context.Context, client opengovernance.Client, connectors []integration.Type, metricIDs []string, startTime time.Time, endTime time.Time, size int) (map[string]SpendMetricResp, error) {
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

	includeConnectorMap := make(map[integration.Type]bool)
	for _, connector := range connectors {
		includeConnectorMap[connector] = true
	}

	b, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	query := string(b)
	fmt.Println("FetchSpendByMetricConnector =", query)
	var response FetchSpendByMetricConnectorQueryResponse
	err = client.Search(ctx, spend.AnalyticsSpendIntegrationTypeSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	resp := map[string]SpendMetricResp{}
	for _, metricBucket := range response.Aggregations.MetricIDGroup.Buckets {
		for _, v := range metricBucket.HitSelect.Hits.Hits {
			if len(connectors) == 0 {
				metricResp := resp[metricBucket.Key]
				metricResp.MetricName = v.Source.MetricName
				metricResp.CostValue += v.Source.TotalCostValue
				resp[metricBucket.Key] = metricResp
				continue
			}
			for _, connectorResult := range v.Source.IntegrationTypes {
				if len(connectors) > 0 && !includeConnectorMap[connectorResult.Connector] {
					continue
				}
				metricResp := resp[metricBucket.Key]
				metricResp.MetricName = v.Source.MetricName
				metricResp.CostValue += connectorResult.CostValue
				resp[metricBucket.Key] = metricResp
			}
		}
	}
	return resp, nil
}

type DimensionTrend struct {
	DimensionID     string
	IntegrationType integration.Type
	DimensionName   string
	Trend           map[string]float64
}

type SpendTableByDimensionQueryResponse struct {
	Aggregations struct {
		DateGroup struct {
			Buckets []struct {
				Key       string `json:"key"`
				HitSelect struct {
					Hits struct {
						Hits []struct {
							Source spend.IntegrationMetricTrendSummary `json:"_source"`
						} `json:"hits"`
					} `json:"hits"`
				} `json:"hit_select"`
			} `json:"buckets"`
		} `json:"date_group"`
	} `json:"aggregations"`
}

func FetchSpendTableByDimension(ctx context.Context, client opengovernance.Client, dimension inventoryAPI.DimensionType, connectionIds []string, connectors []integration.Type, metricIds []string, startTime, endTime time.Time) ([]DimensionTrend, error) {
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
		"date_group": map[string]any{
			"terms": map[string]any{
				"field": "date",
				"size":  EsFetchPageSize,
			},
			"aggs": map[string]any{
				"hit_select": map[string]any{
					"top_hits": map[string]any{
						"size": EsFetchPageSize,
					},
				},
			},
		},
	}

	includeConnectionMap := make(map[string]bool)
	for _, connectionID := range connectionIds {
		includeConnectionMap[connectionID] = true
	}

	includeConnectorMap := make(map[integration.Type]bool)
	for _, connector := range connectors {
		includeConnectorMap[connector] = true
	}

	queryJson, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}
	fmt.Printf("FetchSpendTableByDimension = %s\n", queryJson)

	var response SpendTableByDimensionQueryResponse
	err = client.Search(ctx, spend.AnalyticsSpendIntegrationSummaryIndex, string(queryJson), &response)
	if err != nil {
		return nil, err
	}

	result := make(map[string]DimensionTrend)
	fmt.Println(response)

	for _, dateBucket := range response.Aggregations.DateGroup.Buckets {
		for _, hit := range dateBucket.HitSelect.Hits.Hits {
			for _, connectionResult := range hit.Source.Integrations {
				if (len(connectionIds) > 0 && !includeConnectionMap[connectionResult.IntegrationID]) ||
					(len(connectors) > 0 && !includeConnectorMap[connectionResult.IntegrationType]) {
					continue
				}
				key := ""
				switch dimension {
				case inventoryAPI.DimensionTypeConnection:
					key = connectionResult.IntegrationID
				case inventoryAPI.DimensionTypeMetric:
					key = hit.Source.MetricID
				}
				mt, ok := result[key]
				if !ok {
					mt = DimensionTrend{
						DimensionID:     key,
						IntegrationType: connectionResult.IntegrationType,
						Trend:           make(map[string]float64),
					}
					switch dimension {
					case inventoryAPI.DimensionTypeConnection:
						mt.DimensionName = connectionResult.IntegrationName
					case inventoryAPI.DimensionTypeMetric:
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

type CountAnalyticsSpendResponse struct {
	Aggregations struct {
		MetricCount struct {
			Value int `json:"value"`
		} `json:"metric_count"`
		IntegrationCount struct {
			Value int `json:"value"`
		} `json:"integration_count"`
	} `json:"aggregations"`
}

func CountAnalyticsSpend(ctx context.Context, logger *zap.Logger, client opengovernance.Client) (*CountAnalyticsSpendResponse, error) {
	query := make(map[string]any)
	query["size"] = 0

	integrationScript := `
String[] res = new String[params['_source']['connections'].length];
for (int i=0; i<params['_source']['connections'].length;++i) { 
  res[i] = params['_source']['connections'][i]['connection_id'];
} 
return res;
`
	query["aggs"] = map[string]any{
		"metric_count": map[string]any{
			"cardinality": map[string]any{
				"field": "metric_id",
			},
		},
		"integration_count": map[string]any{
			"cardinality": map[string]any{
				"script": map[string]any{
					"lang":   "painless",
					"source": integrationScript,
				},
			},
		},
	}

	queryJson, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}
	logger.Info("CountAnalyticsSpend", zap.String("query", string(queryJson)))

	var response CountAnalyticsSpendResponse
	err = client.Search(ctx, spend.AnalyticsSpendIntegrationSummaryIndex, string(queryJson), &response)
	if err != nil {
		logger.Error("CountAnalyticsSpend", zap.Error(err), zap.String("query", string(queryJson)))
		return nil, err
	}

	return &response, nil
}
