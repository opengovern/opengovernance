package es

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	summarizer "github.com/kaytu-io/kaytu-engine/pkg/summarizer/es"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type FetchCostHistoryByAccountsQueryResponse struct {
	Aggregations struct {
		ConnectionIDGroup struct {
			Buckets []struct {
				Key          string `json:"key"`
				CostSumGroup struct {
					Value float64 `json:"value"`
				} `json:"cost_sum_group"`
			} `json:"buckets"`
		} `json:"connection_id_group"`
	} `json:"aggregations"`
}

func FetchDailyCostHistoryByAccountsBetween(client kaytu.Client, connectors []source.Type, connectionIDs []string, before time.Time, after time.Time, size int) (map[string]float64, error) {
	before = before.Truncate(24 * time.Hour)
	after = after.Truncate(24 * time.Hour)

	hits := make(map[string]float64)
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

	res["size"] = 0
	res["query"] = map[string]any{
		"bool": map[string]any{
			"filter": filters,
		},
	}
	res["aggs"] = map[string]any{
		"connection_id_group": map[string]any{
			"terms": map[string]any{
				"field": "source_id",
				"size":  size,
			},
			"aggs": map[string]any{
				"cost_sum_group": map[string]any{
					"sum": map[string]string{
						"field": "cost_value",
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
	var response FetchCostHistoryByAccountsQueryResponse
	err = client.Search(context.Background(), summarizer.CostSummeryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	for _, connectionIDGroup := range response.Aggregations.ConnectionIDGroup.Buckets {
		hits[connectionIDGroup.Key] = connectionIDGroup.CostSumGroup.Value
	}

	return hits, nil
}

type FetchDailyCostHistoryByAccountsAtTimeResponse struct {
	Aggregations struct {
		ConnectionIDGroup struct {
			Buckets []struct {
				Key    string `json:"key"`
				Latest struct {
					Hits struct {
						Hits []struct {
							CostSummary summarizer.ConnectionCostSummary `json:"_source"`
						} `json:"hits"`
					} `json:"hits"`
				} `json:"latest"`
			} `json:"buckets"`
		} `json:"connection_id_group"`
	} `json:"aggregations"`
}

func FetchDailyCostHistoryByAccountsAtTime(client kaytu.Client, connectors []source.Type, connectionIDs []string, at time.Time) (map[string]float64, error) {
	at = at.Truncate(24 * time.Hour)

	res := make(map[string]any)
	var filters []any

	filters = append(filters, map[string]any{
		"terms": map[string][]string{"report_type": {string(summarizer.CostConnectionSummaryDaily)}},
	})
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"period_end": map[string]string{
				"lte": strconv.FormatInt(at.Unix(), 10),
			},
		},
	})
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"period_start": map[string]string{
				"gte": strconv.FormatInt(at.AddDate(0, 0, -7).Unix(), 10),
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
		"connection_id_group": map[string]any{
			"terms": map[string]any{
				"field": "source_id",
				"size":  10000,
			},
			"aggs": map[string]any{
				"latest": map[string]any{
					"top_hits": map[string]any{
						"size": 1,
						"sort": map[string]any{
							"period_end": "desc",
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

	fmt.Println("query=", string(b), "index=", summarizer.CostSummeryIndex)

	var response FetchDailyCostHistoryByAccountsAtTimeResponse
	err = client.Search(context.Background(), summarizer.CostSummeryIndex, string(b), &response)
	if err != nil {
		return nil, err
	}

	hits := make(map[string]float64)
	for _, connectionIDGroup := range response.Aggregations.ConnectionIDGroup.Buckets {
		for _, hit := range connectionIDGroup.Latest.Hits.Hits {
			hits[connectionIDGroup.Key] += hit.CostSummary.CostValue
		}
	}

	return hits, nil
}

type ConnectionCostSummaryQueryResponse struct {
	Hits  ConnectionCostSummaryQueryHits `json:"hits"`
	PitID string                         `json:"pit_id"`
}
type ConnectionCostSummaryQueryHits struct {
	Total kaytu.SearchTotal               `json:"total"`
	Hits  []ConnectionCostSummaryQueryHit `json:"hits"`
}
type ConnectionCostSummaryQueryHit struct {
	ID      string                        `json:"_id"`
	Score   float64                       `json:"_score"`
	Index   string                        `json:"_index"`
	Type    string                        `json:"_type"`
	Version int64                         `json:"_version,omitempty"`
	Source  summarizer.ServiceCostSummary `json:"_source"`
	Sort    []any                         `json:"sort"`
}

type ConnectionCostPaginator struct {
	paginator *kaytu.BaseESPaginator
}

func NewConnectionCostPaginator(client kaytu.Client, filters []kaytu.BoolFilter, limit *int64) (ConnectionCostPaginator, error) {
	paginator, err := kaytu.NewPaginator(client.ES(), summarizer.CostSummeryIndex, filters, limit)
	if err != nil {
		return ConnectionCostPaginator{}, err
	}

	p := ConnectionCostPaginator{
		paginator: paginator,
	}

	return p, nil
}

func (p ConnectionCostPaginator) HasNext() bool {
	return !p.paginator.Done()
}

func (p ConnectionCostPaginator) NextPage(ctx context.Context) ([]summarizer.ServiceCostSummary, error) {
	var response ConnectionCostSummaryQueryResponse
	err := p.paginator.Search(ctx, &response)
	if err != nil {
		return nil, err
	}

	var values []summarizer.ServiceCostSummary
	for _, hit := range response.Hits.Hits {
		values = append(values, hit.Source)
	}

	hits := int64(len(response.Hits.Hits))
	if hits > 0 {
		p.paginator.UpdateState(hits, response.Hits.Hits[hits-1].Sort, response.PitID)
	} else {
		p.paginator.UpdateState(hits, nil, "")
	}

	return values, nil
}
