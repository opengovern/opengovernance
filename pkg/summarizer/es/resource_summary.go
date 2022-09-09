package es

import (
	"context"
	"encoding/json"
	"fmt"

	"gitlab.com/keibiengine/keibi-engine/pkg/describe/kafka"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
)

const (
	EsFetchPageSize        = 10000
	SourceResourcesSummary = "source_resources_summary"
)

type ResourceSummaryQueryResponse struct {
	Hits ResourceSummaryQueryHits `json:"hits"`
}
type ResourceSummaryQueryHits struct {
	Total keibi.SearchTotal         `json:"total"`
	Hits  []ResourceSummaryQueryHit `json:"hits"`
}
type ResourceSummaryQueryHit struct {
	ID      string                       `json:"_id"`
	Score   float64                      `json:"_score"`
	Index   string                       `json:"_index"`
	Type    string                       `json:"_type"`
	Version int64                        `json:"_version,omitempty"`
	Source  kafka.SourceResourcesSummary `json:"_source"`
	Sort    []interface{}                `json:"sort"`
}

func FetchResourceSummary(client keibi.Client, jobID uint, sourceID *string) ([]kafka.SourceResourcesSummary, error) {
	var hits []kafka.SourceResourcesSummary
	var searchAfter []interface{}
	for {
		res := make(map[string]interface{})
		var filters []interface{}

		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"report_type": {kafka.ResourceSummaryTypeResourceGrowthTrend}},
		})

		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_job_id": {fmt.Sprintf("%d", jobID)}},
		})

		if sourceID != nil {
			filters = append(filters, map[string]interface{}{
				"terms": map[string][]string{"source_id": {*sourceID}},
			})
		}

		if searchAfter != nil {
			res["search_after"] = searchAfter
		}

		res["size"] = EsFetchPageSize
		res["sort"] = []map[string]interface{}{
			{
				"_id": "desc",
			},
		}
		res["query"] = map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": filters,
			},
		}
		b, err := json.Marshal(res)
		if err != nil {
			return nil, err
		}

		query := string(b)

		var response ResourceSummaryQueryResponse
		err = client.Search(context.Background(), SourceResourcesSummary, query, &response)
		if err != nil {
			return nil, err
		}

		if len(response.Hits.Hits) == 0 {
			break
		}

		for _, hit := range response.Hits.Hits {
			searchAfter = hit.Sort
			hits = append(hits, hit.Source)
		}
	}
	return hits, nil
}

type ServicesSummaryQueryResponse struct {
	Hits ServicesSummaryQueryHits `json:"hits"`
}
type ServicesSummaryQueryHits struct {
	Total keibi.SearchTotal         `json:"total"`
	Hits  []ServicesSummaryQueryHit `json:"hits"`
}
type ServicesSummaryQueryHit struct {
	ID      string                      `json:"_id"`
	Score   float64                     `json:"_score"`
	Index   string                      `json:"_index"`
	Type    string                      `json:"_type"`
	Version int64                       `json:"_version,omitempty"`
	Source  kafka.SourceServicesSummary `json:"_source"`
	Sort    []interface{}               `json:"sort"`
}

func FetchServicesSummary(client keibi.Client, sourceJobIDs []uint) ([]kafka.SourceServicesSummary, error) {
	var hits []kafka.SourceServicesSummary
	var searchAfter []interface{}
	for {
		res := make(map[string]interface{})
		var filters []interface{}

		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"report_type": {kafka.ResourceSummaryTypeServiceHistorySummary}},
		})

		filters = append(filters, map[string]interface{}{
			"terms": map[string][]uint{"source_job_id": sourceJobIDs},
		})

		res["size"] = EsFetchPageSize
		if searchAfter != nil {
			res["search_after"] = searchAfter
		}

		res["sort"] = []map[string]interface{}{
			{
				"_id": "desc",
			},
		}
		res["query"] = map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": filters,
			},
		}
		b, err := json.Marshal(res)
		if err != nil {
			return nil, err
		}

		query := string(b)

		var response ServicesSummaryQueryResponse
		err = client.Search(context.Background(), SourceResourcesSummary, query, &response)
		if err != nil {
			return nil, err
		}

		if len(response.Hits.Hits) == 0 {
			break
		}

		for _, hit := range response.Hits.Hits {
			hits = append(hits, hit.Source)
			searchAfter = hit.Sort
		}
	}
	return hits, nil
}

type CategoriesSummaryQueryResponse struct {
	Hits CategoriesSummaryQueryHits `json:"hits"`
}
type CategoriesSummaryQueryHits struct {
	Total keibi.SearchTotal           `json:"total"`
	Hits  []CategoriesSummaryQueryHit `json:"hits"`
}
type CategoriesSummaryQueryHit struct {
	ID      string                      `json:"_id"`
	Score   float64                     `json:"_score"`
	Index   string                      `json:"_index"`
	Type    string                      `json:"_type"`
	Version int64                       `json:"_version,omitempty"`
	Source  kafka.SourceCategorySummary `json:"_source"`
	Sort    []interface{}               `json:"sort"`
}

func FetchCategoriesSummary(client keibi.Client, sourceJobIDs []uint) ([]kafka.SourceCategorySummary, error) {
	var hits []kafka.SourceCategorySummary
	var searchAfter []interface{}
	for {
		res := make(map[string]interface{})
		var filters []interface{}

		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"report_type": {kafka.ResourceSummaryTypeCategoryHistorySummary}},
		})

		filters = append(filters, map[string]interface{}{
			"terms": map[string][]uint{"source_job_id": sourceJobIDs},
		})

		res["size"] = EsFetchPageSize
		if searchAfter != nil {
			res["search_after"] = searchAfter
		}

		res["sort"] = []map[string]interface{}{
			{
				"_id": "desc",
			},
		}
		res["query"] = map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": filters,
			},
		}
		b, err := json.Marshal(res)
		if err != nil {
			return nil, err
		}

		query := string(b)

		var response CategoriesSummaryQueryResponse
		err = client.Search(context.Background(), SourceResourcesSummary, query, &response)
		if err != nil {
			return nil, err
		}

		if len(response.Hits.Hits) == 0 {
			break
		}

		for _, hit := range response.Hits.Hits {
			hits = append(hits, hit.Source)
			searchAfter = hit.Sort
		}
	}
	return hits, nil
}

type LocationSummaryQueryResponse struct {
	Hits LocationSummaryQueryHits `json:"hits"`
}
type LocationSummaryQueryHits struct {
	Total keibi.SearchTotal         `json:"total"`
	Hits  []LocationSummaryQueryHit `json:"hits"`
}
type LocationSummaryQueryHit struct {
	ID      string                             `json:"_id"`
	Score   float64                            `json:"_score"`
	Index   string                             `json:"_index"`
	Type    string                             `json:"_type"`
	Version int64                              `json:"_version,omitempty"`
	Source  kafka.LocationDistributionResource `json:"_source"`
	Sort    []interface{}                      `json:"sort"`
}

func FetchLocationSummary(client keibi.Client, jobID uint, sourceID *string) ([]kafka.LocationDistributionResource, error) {
	var hits []kafka.LocationDistributionResource
	var searchAfter []interface{}
	for {
		res := make(map[string]interface{})
		var filters []interface{}

		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"report_type": {kafka.ResourceSummaryTypeLocationDistribution}},
		})

		filters = append(filters, map[string]interface{}{
			"terms": map[string][]string{"source_job_id": {fmt.Sprintf("%d", jobID)}},
		})

		if sourceID != nil {
			filters = append(filters, map[string]interface{}{
				"terms": map[string][]string{"source_id": {*sourceID}},
			})
		}

		if searchAfter != nil {
			res["search_after"] = searchAfter
		}

		res["size"] = EsFetchPageSize
		res["sort"] = []map[string]interface{}{
			{
				"_id": "desc",
			},
		}
		res["query"] = map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": filters,
			},
		}
		b, err := json.Marshal(res)
		if err != nil {
			return nil, err
		}

		query := string(b)

		var response LocationSummaryQueryResponse
		err = client.Search(context.Background(), SourceResourcesSummary, query, &response)
		if err != nil {
			return nil, err
		}

		if len(response.Hits.Hits) == 0 {
			break
		}

		for _, hit := range response.Hits.Hits {
			searchAfter = hit.Sort
			hits = append(hits, hit.Source)
		}
	}
	return hits, nil
}
