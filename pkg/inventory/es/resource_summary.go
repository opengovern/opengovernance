package es

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/kaytu-io/kaytu-util/pkg/source"
	"gitlab.com/keibiengine/keibi-engine/pkg/utils"

	summarizer "gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"

	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/api"

	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
)

func FindAWSCostQuery(sourceID *uuid.UUID, fetchSize int, searchAfter []any) (string, error) {
	res := make(map[string]any)

	res["size"] = fetchSize
	res["sort"] = []map[string]any{
		{
			"_id": "asc",
		},
	}
	if searchAfter != nil {
		res["search_after"] = searchAfter
	}

	if sourceID != nil {
		var filters []any
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"source_id": {sourceID.String()}},
		})
		res["query"] = map[string]any{
			"bool": map[string]any{
				"filter": filters,
			},
		}
	}
	b, err := json.Marshal(res)
	return string(b), err
}

type LookupResourceAggregationResponse struct {
	Aggregations LookupResourceAggregations `json:"aggregations"`
}
type LookupResourceAggregations struct {
	ResourceTypeFilter AggregationResult `json:"resource_type_filter"`
	SourceTypeFilter   AggregationResult `json:"source_type_filter"`
	LocationFilter     AggregationResult `json:"location_filter"`
	ConnectionFilter   AggregationResult `json:"connection_id_filter"`
}
type AggregationResult struct {
	DocCountErrorUpperBound int      `json:"doc_count_error_upper_bound"`
	SumOtherDocCount        int      `json:"sum_other_doc_count"`
	Buckets                 []Bucket `json:"buckets"`
}
type Bucket struct {
	Key      string `json:"key"`
	DocCount int    `json:"doc_count"`
}

func BuildFilterQuery(
	query string,
	filters api.ResourceFilters,
	commonFilter *bool,
) (string, error) {
	terms := make(map[string][]string)
	if !api.FilterIsEmpty(filters.Location) {
		terms["location"] = filters.Location
	}

	if !api.FilterIsEmpty(filters.ResourceType) {
		filters.ResourceType = utils.ToLowerStringSlice(filters.ResourceType)
		terms["resource_type"] = filters.ResourceType
	}

	if !api.FilterIsEmpty(filters.Provider) {
		terms["source_type"] = filters.Provider
	}

	if !api.FilterIsEmpty(filters.Connections) {
		terms["source_id"] = filters.Connections
	}

	if commonFilter != nil {
		terms["is_common"] = []string{fmt.Sprintf("%v", *commonFilter)}
	}

	notTerms := make(map[string][]string)
	ignoreResourceTypes := []string{
		"Microsoft.Resources/subscriptions/locations",
		"Microsoft.Authorization/roleDefinitions",
		"microsoft.security/autoProvisioningSettings",
		"microsoft.security/settings",
		"Microsoft.Authorization/elevateAccessRoleAssignment",
		"Microsoft.AppConfiguration/configurationStores",
		"Microsoft.KeyVault/vaults/keys",
		"microsoft.security/pricings",
		"Microsoft.Security/autoProvisioningSettings",
		"Microsoft.Security/securityContacts",
		"Microsoft.Security/locations/jitNetworkAccessPolicies",
		"AWS::EC2::Region",
		"AWS::EC2::RegionalSettings",
	}
	notTerms["resource_type"] = utils.ToLowerStringSlice(ignoreResourceTypes)

	root := map[string]any{}
	root["size"] = 0

	sourceTypeFilter := map[string]any{
		"terms": map[string]any{"field": "source_type", "size": 1000},
	}
	resourceTypeFilter := map[string]any{
		"terms": map[string]any{"field": "resource_type", "size": 1000},
	}
	locationFilter := map[string]any{
		"terms": map[string]any{"field": "location", "size": 1000},
	}
	connectionIDFilter := map[string]any{
		"terms": map[string]any{"field": "source_id", "size": 1000},
	}
	aggs := map[string]any{
		"source_type_filter":   sourceTypeFilter,
		"resource_type_filter": resourceTypeFilter,
		"location_filter":      locationFilter,
		"connection_id_filter": connectionIDFilter,
	}
	root["aggs"] = aggs

	boolQuery := make(map[string]any)
	if terms != nil && len(terms) > 0 {
		var filters []map[string]any
		for k, vs := range terms {
			filters = append(filters, map[string]any{
				"terms": map[string][]string{
					k: vs,
				},
			})
		}

		boolQuery["filter"] = filters
	}
	if len(query) > 0 {
		boolQuery["must"] = map[string]any{
			"multi_match": map[string]any{
				"fields": []string{"resource_id", "name", "source_type", "resource_type", "resource_group",
					"location", "source_id"},
				"query":     query,
				"fuzziness": "AUTO",
			},
		}
	}
	if len(notTerms) > 0 {
		var filters []map[string]any
		for k, vs := range notTerms {
			filters = append(filters, map[string]any{
				"terms": map[string][]string{
					k: vs,
				},
			})
		}

		boolQuery["must_not"] = filters
	}
	if len(boolQuery) > 0 {
		root["query"] = map[string]any{
			"bool": boolQuery,
		}
	}

	queryBytes, err := json.Marshal(root)
	if err != nil {
		return "", err
	}
	return string(queryBytes), nil
}

type ConnectionResourcesSummaryQueryResponse struct {
	Hits ConnectionResourcesSummaryQueryHits `json:"hits"`
}
type ConnectionResourcesSummaryQueryHits struct {
	Total keibi.SearchTotal                    `json:"total"`
	Hits  []ConnectionResourcesSummaryQueryHit `json:"hits"`
}
type ConnectionResourcesSummaryQueryHit struct {
	ID      string                                `json:"_id"`
	Score   float64                               `json:"_score"`
	Index   string                                `json:"_index"`
	Type    string                                `json:"_type"`
	Version int64                                 `json:"_version,omitempty"`
	Source  summarizer.ConnectionResourcesSummary `json:"_source"`
	Sort    []any                                 `json:"sort"`
}

func FetchConnectionResourcesSummaryPage(client keibi.Client, connectors []source.Type, sourceID []string, sort []map[string]any, size int) ([]summarizer.ConnectionResourcesSummary, error) {
	var hits []summarizer.ConnectionResourcesSummary
	res := make(map[string]any)
	var filters []any

	filters = append(filters, map[string]any{
		"terms": map[string][]string{"report_type": {string(summarizer.ResourceSummary)}},
	})

	if len(connectors) > 0 {
		connectorsStr := make([]string, 0, len(connectors))
		for _, c := range connectors {
			connectorsStr = append(connectorsStr, c.String())
		}
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"source_type": connectorsStr},
		})
	}

	if sourceID != nil && len(sourceID) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"source_id": sourceID},
		})
	}

	sort = append(sort,
		map[string]any{
			"_id": "desc",
		},
	)
	res["size"] = size
	res["sort"] = sort
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

	var response ConnectionResourcesSummaryQueryResponse
	err = client.Search(context.Background(), summarizer.ConnectionSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	for _, hit := range response.Hits.Hits {
		hits = append(hits, hit.Source)
	}
	return hits, nil
}

type FetchConnectionResourcesCountAtResponse struct {
	Aggregations struct {
		ConnectionIDGroup struct {
			Key     string `json:"key"`
			Buckets []struct {
				Latest struct {
					Hits ConnectionResourcesSummaryQueryHits `json:"hits"`
				} `json:"latest"`
			} `json:"buckets"`
		} `json:"connection_id_group"`
	} `json:"aggregations"`
}

func FetchConnectionResourcesCountAtTime(client keibi.Client, connectors []source.Type, connectionIDs []string, t time.Time, size int) ([]summarizer.ConnectionResourcesSummary, error) {
	var hits []summarizer.ConnectionResourcesSummary
	res := make(map[string]any)
	var filters []any

	t = t.Truncate(24 * time.Hour)

	filters = append(filters, map[string]any{
		"terms": map[string][]string{"report_type": {string(summarizer.ResourceSummary) + "History"}},
	})

	filters = append(filters, map[string]any{
		"range": map[string]any{
			"described_at": map[string]string{
				"lte": strconv.FormatInt(t.UnixMilli(), 10),
			},
		},
	})

	if len(connectors) > 0 {
		connectorsStr := make([]string, 0, len(connectors))
		for _, c := range connectors {
			connectorsStr = append(connectorsStr, c.String())
		}
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"source_type": connectorsStr},
		})
	}

	if len(connectionIDs) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"source_id": connectionIDs},
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
				"latest": map[string]any{
					"top_hits": map[string]any{
						"size": 1,
						"sort": map[string]string{
							"described_at": "desc",
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
	fmt.Println("query=", query, "index=", summarizer.ConnectionSummaryIndex)
	var response FetchConnectionResourcesCountAtResponse
	err = client.Search(context.Background(), summarizer.ConnectionSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	for _, connectionIdBucket := range response.Aggregations.ConnectionIDGroup.Buckets {
		for _, hit := range connectionIdBucket.Latest.Hits.Hits {
			hits = append(hits, hit.Source)
		}
	}
	return hits, nil
}

type ConnectionLocationsSummaryQueryResponse struct {
	Aggregations struct {
		ConnectionIdGroup struct {
			Buckets []struct {
				Key    string `json:"key"`
				Latest struct {
					Hits struct {
						Hits []struct {
							Source summarizer.ConnectionLocationSummary `json:"_source"`
						} `json:"hits"`
					} `json:"hits"`
				} `json:"latest"`
			} `json:"buckets"`
		} `json:"connection_id_group"`
	} `json:"aggregations"`
}

func FetchConnectionLocationsSummaryPage(client keibi.Client, connectors []source.Type, connectionIDs []string, sort []map[string]any, timeAt time.Time) ([]summarizer.ConnectionLocationSummary, error) {
	var hits []summarizer.ConnectionLocationSummary
	res := make(map[string]any)

	var filters []any
	filters = append(filters, map[string]any{
		"terms": map[string][]string{"report_type": {string(summarizer.LocationConnectionSummaryHistory)}},
	})
	if len(connectors) > 0 {
		connectorStr := make([]string, 0, len(connectors))
		for _, connector := range connectors {
			connectorStr = append(connectorStr, connector.String())
		}
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"source_type": connectorStr},
		})
	}
	if len(connectionIDs) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"source_id": connectionIDs},
		})
	}
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"summarized_at": map[string]any{
				"lte": timeAt.Unix(),
			},
		},
	})
	res["size"] = 0
	if sort != nil {
		res["sort"] = sort
	}
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
						"sort": map[string]string{
							"summarized_at": "desc",
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

	fmt.Printf("query= %s, index= %s\n", query, summarizer.ConnectionSummaryIndex)

	var response ConnectionLocationsSummaryQueryResponse
	err = client.Search(context.Background(), summarizer.ConnectionSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	for _, connectionIdBucket := range response.Aggregations.ConnectionIdGroup.Buckets {
		for _, hit := range connectionIdBucket.Latest.Hits.Hits {
			hits = append(hits, hit.Source)
		}
	}
	return hits, nil
}

type ConnectionTrendSummaryQueryResponse struct {
	Hits ConnectionTrendSummaryQueryHits `json:"hits"`
}
type ConnectionTrendSummaryQueryHits struct {
	Total keibi.SearchTotal                `json:"total"`
	Hits  []ConnectionTrendSummaryQueryHit `json:"hits"`
}
type ConnectionTrendSummaryQueryHit struct {
	ID      string                            `json:"_id"`
	Score   float64                           `json:"_score"`
	Index   string                            `json:"_index"`
	Type    string                            `json:"_type"`
	Version int64                             `json:"_version,omitempty"`
	Source  summarizer.ConnectionTrendSummary `json:"_source"`
	Sort    []any                             `json:"sort"`
}

func FetchConnectionTrendSummaryPage(client keibi.Client, connectionIDs []string, createdAtFrom, createdAtTo int64,
	sort []map[string]any, size int) ([]summarizer.ConnectionTrendSummary, error) {
	var hits []summarizer.ConnectionTrendSummary
	res := make(map[string]any)
	var filters []any

	filters = append(filters, map[string]any{
		"terms": map[string][]string{"report_type": {string(summarizer.TrendConnectionSummary)}},
	})

	if len(connectionIDs) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"source_id": connectionIDs},
		})
	}
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"described_at": map[string]string{
				"gte": strconv.FormatInt(createdAtFrom, 10),
				"lte": strconv.FormatInt(createdAtTo, 10),
			},
		},
	})

	sort = append(sort,
		map[string]any{
			"_id": "desc",
		},
	)
	res["size"] = size
	res["sort"] = sort
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

	var response ConnectionTrendSummaryQueryResponse
	err = client.Search(context.Background(), summarizer.ConnectionSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	for _, hit := range response.Hits.Hits {
		hits = append(hits, hit.Source)
	}
	return hits, nil
}

type ConnectionResourceTypeTrendSummaryQueryResponse struct {
	Aggregations struct {
		ResourceTypeGroup struct {
			Buckets []struct {
				Key                   string `json:"key"`
				DescribedAtRangeGroup struct {
					Buckets []struct {
						From   float64 `json:"from"`
						To     float64 `json:"to"`
						Latest struct {
							Hits struct {
								Hits []struct {
									Source summarizer.ConnectionResourceTypeTrendSummary `json:"_source"`
								} `json:"hits"`
							} `json:"hits"`
						} `json:"latest"`
					} `json:"buckets"`
				} `json:"described_at_range_group"`
			} `json:"buckets"`
		} `json:"resource_type_group"`
	} `json:"aggregations"`
}

func FetchConnectionResourceTypeTrendSummaryPage(client keibi.Client, connectionIDs, resourceTypes []string, startTime, endTime time.Time, datapointCount int, size int) (map[int]int, error) {
	res := make(map[string]any)
	var filters []any

	filters = append(filters, map[string]any{
		"terms": map[string][]string{"report_type": {string(summarizer.ResourceTypeTrendConnectionSummary)}},
	})
	resourceTypes = utils.ToLowerStringSlice(resourceTypes)
	filters = append(filters, map[string]any{
		"terms": map[string][]string{"resource_type": resourceTypes},
	})
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"described_at": map[string]string{
				"gte": strconv.FormatInt(startTime.UnixMilli(), 10),
				"lte": strconv.FormatInt(endTime.UnixMilli(), 10),
			},
		},
	})
	res["size"] = 0
	startTimeUnixMilli := startTime.UnixMilli()
	endTimeUnixMilli := endTime.UnixMilli()
	step := int(math.Ceil(float64(endTimeUnixMilli-startTimeUnixMilli) / float64(datapointCount)))
	ranges := make([]map[string]any, 0, datapointCount)
	for i := 0; i < datapointCount; i++ {
		ranges = append(ranges, map[string]any{
			"from": float64(startTimeUnixMilli + int64(step*i)),
			"to":   float64(startTimeUnixMilli + int64(step*(i+1))),
		})
	}
	res["aggs"] = map[string]any{
		"resource_type_group": map[string]any{
			"terms": map[string]any{
				"field": "resource_type",
				"size":  size,
			},
			"aggs": map[string]any{
				"described_at_range_group": map[string]any{
					"range": map[string]any{
						"field":  "described_at",
						"ranges": ranges,
					},
					"aggs": map[string]any{
						"latest": map[string]any{
							"top_hits": map[string]any{
								"size": 1,
								"sort": map[string]string{
									"described_at": "desc",
								},
							},
						},
					},
				},
			},
		},
	}

	hits := make(map[int]int)
	for _, connectionID := range connectionIDs {
		localFilters := append(filters, map[string]any{
			"term": map[string]string{"source_id": connectionID},
		})
		res["query"] = map[string]any{
			"bool": map[string]any{
				"filter": localFilters,
			},
		}

		b, err := json.Marshal(res)
		if err != nil {
			return nil, err
		}
		query := string(b)
		fmt.Println("query=", query, "index=", summarizer.ConnectionSummaryIndex)
		var response ConnectionResourceTypeTrendSummaryQueryResponse
		err = client.Search(context.Background(), summarizer.ConnectionSummaryIndex, query, &response)
		if err != nil {
			return nil, err
		}
		for _, resourceTypeBucket := range response.Aggregations.ResourceTypeGroup.Buckets {
			for _, describedAtRangeBucket := range resourceTypeBucket.DescribedAtRangeGroup.Buckets {
				rangeKey := int((describedAtRangeBucket.From + describedAtRangeBucket.To) / 2)
				for _, hit := range describedAtRangeBucket.Latest.Hits.Hits {
					hits[rangeKey] += hit.Source.ResourceCount
				}
			}
		}
	}

	return hits, nil
}

type ProviderTrendSummaryQueryResponse struct {
	Hits ProviderTrendSummaryQueryHits `json:"hits"`
}
type ProviderTrendSummaryQueryHits struct {
	Total keibi.SearchTotal              `json:"total"`
	Hits  []ProviderTrendSummaryQueryHit `json:"hits"`
}
type ProviderTrendSummaryQueryHit struct {
	ID      string                          `json:"_id"`
	Score   float64                         `json:"_score"`
	Index   string                          `json:"_index"`
	Type    string                          `json:"_type"`
	Version int64                           `json:"_version,omitempty"`
	Source  summarizer.ProviderTrendSummary `json:"_source"`
	Sort    []any                           `json:"sort"`
}

func FetchProviderTrendSummaryPage(client keibi.Client, connectors []source.Type, createdAtFrom, createdAtTo int64,
	sort []map[string]any, size int) ([]summarizer.ProviderTrendSummary, error) {
	var hits []summarizer.ProviderTrendSummary
	res := make(map[string]any)
	var filters []any

	filters = append(filters, map[string]any{
		"terms": map[string][]string{"report_type": {string(summarizer.TrendProviderSummary)}},
	})

	if len(connectors) > 0 {
		connectorsStr := make([]string, 0, len(connectors))
		for _, connector := range connectors {
			connectorsStr = append(connectorsStr, string(connector))
		}
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"source_type": connectorsStr},
		})
	}
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"described_at": map[string]string{
				"gte": strconv.FormatInt(createdAtFrom, 10),
				"lte": strconv.FormatInt(createdAtTo, 10),
			},
		},
	})

	sort = append(sort,
		map[string]any{
			"_id": "desc",
		},
	)
	res["size"] = size
	res["sort"] = sort
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
	fmt.Println("query=", query, "index=", summarizer.ProviderSummaryIndex)
	var response ProviderTrendSummaryQueryResponse
	err = client.Search(context.Background(), summarizer.ProviderSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	for _, hit := range response.Hits.Hits {
		hits = append(hits, hit.Source)
	}
	return hits, nil
}

type ProviderResourceTypeTrendSummaryQueryResponse struct {
	Aggregations struct {
		ResourceTypeGroup struct {
			Buckets []struct {
				Key                   string `json:"key"`
				DescribedAtRangeGroup struct {
					Buckets []struct {
						From   float64 `json:"from"`
						To     float64 `json:"to"`
						Latest struct {
							Hits struct {
								Hits []struct {
									Source summarizer.ConnectionResourceTypeTrendSummary `json:"_source"`
								} `json:"hits"`
							} `json:"hits"`
						} `json:"latest"`
					} `json:"buckets"`
				} `json:"described_at_range_group"`
			} `json:"buckets"`
		} `json:"resource_type_group"`
	} `json:"aggregations"`
}

func FetchProviderResourceTypeTrendSummaryPage(client keibi.Client, connectors []source.Type, resourceTypes []string, startTime, endTime time.Time, datapointCount int, size int) (map[int]int, error) {
	res := make(map[string]any)
	var filters []any

	filters = append(filters, map[string]any{
		"terms": map[string][]string{"report_type": {string(summarizer.ResourceTypeTrendProviderSummary)}},
	})

	resourceTypes = utils.ToLowerStringSlice(resourceTypes)
	filters = append(filters, map[string]any{
		"terms": map[string][]string{"resource_type": resourceTypes},
	})

	if len(connectors) > 0 {
		connectorsStr := make([]string, 0, len(connectors))
		for _, connector := range connectors {
			connectorsStr = append(connectorsStr, string(connector))
		}
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"source_type": connectorsStr},
		})
	}
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"described_at": map[string]string{
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

	startTimeUnixMilli := startTime.UnixMilli()
	endTimeUnixMilli := endTime.UnixMilli()
	step := int(math.Ceil(float64(endTimeUnixMilli-startTimeUnixMilli) / float64(datapointCount)))
	ranges := make([]map[string]any, 0, datapointCount)
	for i := 0; i < datapointCount; i++ {
		ranges = append(ranges, map[string]any{
			"from": float64(startTimeUnixMilli + int64(step*i)),
			"to":   float64(startTimeUnixMilli + int64(step*(i+1))),
		})
	}
	res["aggs"] = map[string]any{
		"resource_type_group": map[string]any{
			"terms": map[string]any{
				"field": "resource_type",
				"size":  size,
			},
			"aggs": map[string]any{
				"described_at_range_group": map[string]any{
					"range": map[string]any{
						"field":  "described_at",
						"ranges": ranges,
					},
					"aggs": map[string]any{
						"latest": map[string]any{
							"top_hits": map[string]any{
								"size": 1,
								"sort": map[string]string{
									"described_at": "desc",
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
	fmt.Println("query=", query, "index=", summarizer.ProviderSummaryIndex)
	var response ProviderResourceTypeTrendSummaryQueryResponse
	err = client.Search(context.Background(), summarizer.ProviderSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	hits := make(map[int]int)
	for _, resourceTypeBucket := range response.Aggregations.ResourceTypeGroup.Buckets {
		for _, describedAtRangeBucket := range resourceTypeBucket.DescribedAtRangeGroup.Buckets {
			rangeKey := int((describedAtRangeBucket.From + describedAtRangeBucket.To) / 2)
			for _, hit := range describedAtRangeBucket.Latest.Hits.Hits {
				hits[rangeKey] += hit.Source.ResourceCount
			}
		}
	}

	return hits, nil
}

type ConnectionServiceLocationsSummaryQueryResponse struct {
	Hits ConnectionServiceLocationsSummaryQueryHits `json:"hits"`
}
type ConnectionServiceLocationsSummaryQueryHits struct {
	Total keibi.SearchTotal                           `json:"total"`
	Hits  []ConnectionServiceLocationsSummaryQueryHit `json:"hits"`
}
type ConnectionServiceLocationsSummaryQueryHit struct {
	ID      string                                      `json:"_id"`
	Score   float64                                     `json:"_score"`
	Index   string                                      `json:"_index"`
	Type    string                                      `json:"_type"`
	Version int64                                       `json:"_version,omitempty"`
	Source  summarizer.ConnectionServiceLocationSummary `json:"_source"`
	Sort    []any                                       `json:"sort"`
}

func FetchConnectionServiceLocationsSummaryPage(client keibi.Client, provider source.Type, connectionIDs []string, sort []map[string]any, size int) ([]summarizer.ConnectionServiceLocationSummary, error) {
	var hits []summarizer.ConnectionServiceLocationSummary
	res := make(map[string]any)
	var filters []any

	filters = append(filters, map[string]any{
		"terms": map[string][]string{"report_type": {string(summarizer.ServiceLocationConnectionSummary)}},
	})

	if !provider.IsNull() {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"source_type": {provider.String()}},
		})
	}

	if len(connectionIDs) > 0 {
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"source_id": connectionIDs},
		})
	}

	sort = append(sort,
		map[string]any{
			"_id": "desc",
		},
	)
	res["size"] = size
	res["sort"] = sort
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

	fmt.Println("query=", query, "index=", summarizer.ConnectionSummaryIndex)
	var response ConnectionServiceLocationsSummaryQueryResponse
	err = client.Search(context.Background(), summarizer.ConnectionSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	for _, hit := range response.Hits.Hits {
		hits = append(hits, hit.Source)
	}
	return hits, nil
}

type FetchConnectionResourceTypeCountAtTimeResponse struct {
	Aggregations struct {
		ResourceTypeGroup struct {
			Buckets []struct {
				Key    string `json:"key"`
				Latest struct {
					Hits struct {
						Hits []struct {
							Source summarizer.ConnectionResourceTypeTrendSummary `json:"_source"`
						} `json:"hits"`
					} `json:"hits"`
				} `json:"latest"`
			} `json:"buckets"`
		} `json:"resource_type_group"`
	} `json:"aggregations"`
}

func FetchConnectionResourceTypeCountAtTime(client keibi.Client, connectors []source.Type, connectionIDs []string, t time.Time, resourceTypes []string, size int) (map[string]int, error) {
	res := make(map[string]any)
	var filters []any

	filters = append(filters, map[string]any{
		"terms": map[string][]string{"report_type": {string(summarizer.ResourceTypeTrendConnectionSummary)}},
	})
	resourceTypes = utils.ToLowerStringSlice(resourceTypes)
	filters = append(filters, map[string]any{
		"terms": map[string][]string{"resource_type": resourceTypes},
	})
	if len(connectors) > 0 {
		connectorStrings := make([]string, 0, len(connectors))
		for _, provider := range connectors {
			connectorStrings = append(connectorStrings, provider.String())
		}
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"source_type": connectorStrings},
		})
	}
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"described_at": map[string]string{
				"lte": strconv.FormatInt(t.UnixMilli(), 10),
			},
		},
	})
	res["size"] = 0
	res["aggs"] = map[string]any{
		"resource_type_group": map[string]any{
			"terms": map[string]any{
				"field": "resource_type",
				"size":  size,
			},
			"aggs": map[string]any{
				"latest": map[string]any{
					"top_hits": map[string]any{
						"size": 1,
						"sort": map[string]string{
							"described_at": "desc",
						},
					},
				},
			},
		},
	}

	result := make(map[string]int)
	for _, connectionId := range connectionIDs {
		localFilter := append(filters, map[string]any{
			"term": map[string]string{"source_id": connectionId},
		})
		res["query"] = map[string]any{
			"bool": map[string]any{
				"filter": localFilter,
			},
		}
		b, err := json.Marshal(res)
		if err != nil {
			return nil, err
		}

		query := string(b)
		fmt.Println("query=", query, "index=", summarizer.ConnectionSummaryIndex)
		var response FetchConnectionResourceTypeCountAtTimeResponse
		err = client.Search(context.Background(), summarizer.ConnectionSummaryIndex, query, &response)
		if err != nil {
			return nil, err
		}
		for _, resourceTypeBucket := range response.Aggregations.ResourceTypeGroup.Buckets {
			for _, hit := range resourceTypeBucket.Latest.Hits.Hits {
				result[hit.Source.ResourceType] += hit.Source.ResourceCount
			}
		}
	}

	return result, nil
}

type FetchConnectorResourceTypeCountAtTimeResponse struct {
	Aggregations struct {
		ResourceTypeGroup struct {
			Buckets []struct {
				Key    string `json:"key"`
				Latest struct {
					Hits struct {
						Hits []struct {
							Source summarizer.ProviderResourceTypeTrendSummary `json:"_source"`
						} `json:"hits"`
					} `json:"hits"`
				} `json:"latest"`
			} `json:"buckets"`
		} `json:"resource_type_group"`
	} `json:"aggregations"`
}

func FetchConnectorResourceTypeCountAtTime(client keibi.Client, connectors []source.Type, t time.Time, resourceTypes []string, size int) (map[string]int, error) {
	res := make(map[string]any)
	var filters []any

	filters = append(filters, map[string]any{
		"terms": map[string][]string{"report_type": {string(summarizer.ResourceTypeTrendProviderSummary)}},
	})
	resourceTypes = utils.ToLowerStringSlice(resourceTypes)
	filters = append(filters, map[string]any{
		"terms": map[string][]string{"resource_type": resourceTypes},
	})
	if len(connectors) > 0 {
		connectorStrings := make([]string, 0, len(connectors))
		for _, provider := range connectors {
			connectorStrings = append(connectorStrings, provider.String())
		}
		filters = append(filters, map[string]any{
			"terms": map[string][]string{"source_type": connectorStrings},
		})
	}
	filters = append(filters, map[string]any{
		"range": map[string]any{
			"described_at": map[string]string{
				"lte": strconv.FormatInt(t.UnixMilli(), 10),
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
		"resource_type_group": map[string]any{
			"terms": map[string]any{
				"field": "resource_type",
				"size":  size,
			},
			"aggs": map[string]any{
				"latest": map[string]any{
					"top_hits": map[string]any{
						"size": 1,
						"sort": map[string]string{
							"described_at": "desc",
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
	fmt.Println("query=", query, "index=", summarizer.ProviderSummaryIndex)
	var response FetchConnectorResourceTypeCountAtTimeResponse
	err = client.Search(context.Background(), summarizer.ProviderSummaryIndex, query, &response)
	if err != nil {
		return nil, err
	}

	result := make(map[string]int)
	for _, resourceTypeBucket := range response.Aggregations.ResourceTypeGroup.Buckets {
		for _, hit := range resourceTypeBucket.Latest.Hits.Hits {
			result[hit.Source.ResourceType] += hit.Source.ResourceCount
		}
	}
	return result, nil
}
