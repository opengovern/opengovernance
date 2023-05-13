package resourcebuilder

import (
	"context"
	"encoding/json"
	"fmt"

	"gitlab.com/keibiengine/keibi-engine/pkg/cloudservice"
	describe "gitlab.com/keibiengine/keibi-engine/pkg/describe/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/kafka"
	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"
)

type categorySummaryBuilder struct {
	client            keibi.Client
	summarizerJobID   uint
	connectionSummary map[string]es.ConnectionCategorySummary
	providerSummary   map[string]es.ProviderCategorySummary
}

func NewCategorySummaryBuilder(client keibi.Client, summarizerJobID uint) *categorySummaryBuilder {
	return &categorySummaryBuilder{
		client:            client,
		summarizerJobID:   summarizerJobID,
		connectionSummary: make(map[string]es.ConnectionCategorySummary),
		providerSummary:   make(map[string]es.ProviderCategorySummary),
	}
}

func (b *categorySummaryBuilder) Process(resource describe.LookupResource) {
	key := resource.SourceID + "-" + cloudservice.CategoryByResourceType(resource.ResourceType)
	if _, ok := b.connectionSummary[key]; !ok {
		b.connectionSummary[key] = es.ConnectionCategorySummary{
			SummarizeJobID:   b.summarizerJobID,
			ScheduleJobID:    resource.ScheduleJobID,
			SourceID:         resource.SourceID,
			SourceJobID:      resource.SourceJobID,
			CategoryName:     cloudservice.CategoryByResourceType(resource.ResourceType),
			ResourceType:     resource.ResourceType,
			SourceType:       resource.SourceType,
			DescribedAt:      resource.CreatedAt,
			ResourceCount:    0,
			LastDayCount:     nil,
			LastWeekCount:    nil,
			LastQuarterCount: nil,
			LastYearCount:    nil,
			ReportType:       es.CategorySummary,
		}
	}

	v := b.connectionSummary[key]
	v.ResourceCount++
	b.connectionSummary[key] = v

	key = string(resource.SourceType) + "-" + cloudservice.CategoryByResourceType(resource.ResourceType)
	if _, ok := b.providerSummary[key]; !ok {
		b.providerSummary[key] = es.ProviderCategorySummary{
			SummarizeJobID:   b.summarizerJobID,
			ScheduleJobID:    resource.ScheduleJobID,
			CategoryName:     cloudservice.CategoryByResourceType(resource.ResourceType),
			ResourceType:     resource.ResourceType,
			SourceType:       resource.SourceType,
			DescribedAt:      resource.CreatedAt,
			ResourceCount:    0,
			LastDayCount:     nil,
			LastWeekCount:    nil,
			LastQuarterCount: nil,
			LastYearCount:    nil,
			ReportType:       es.CategoryProviderSummary,
		}
	}

	v2 := b.providerSummary[key]
	v2.ResourceCount++
	b.providerSummary[key] = v2
}

func (b *categorySummaryBuilder) PopulateHistory(lastDayJobID, lastWeekJobID, lastQuarterJobID, lastYearJobID uint) error {
	jobIDs := []uint{lastDayJobID, lastWeekJobID, lastQuarterJobID, lastYearJobID}
	for k, connSummary := range b.connectionSummary {
		for idx, jid := range jobIDs {
			if jid == 0 {
				continue
			}

			r, err := b.queryCategoryConnectionResourceCount(jid, connSummary.SourceID, connSummary.CategoryName)
			if err != nil {
				return err
			}

			switch idx {
			case 0:
				connSummary.LastDayCount = &r
			case 1:
				connSummary.LastWeekCount = &r
			case 2:
				connSummary.LastQuarterCount = &r
			case 3:
				connSummary.LastYearCount = &r
			}
		}
		b.connectionSummary[k] = connSummary
	}

	for k, pSummary := range b.providerSummary {
		for idx, jid := range jobIDs {
			if jid == 0 {
				continue
			}

			r, err := b.queryCategoryProviderResourceCount(jid, pSummary.SourceType, pSummary.CategoryName)
			if err != nil {
				return err
			}

			switch idx {
			case 0:
				pSummary.LastDayCount = &r
			case 1:
				pSummary.LastWeekCount = &r
			case 2:
				pSummary.LastQuarterCount = &r
			case 3:
				pSummary.LastYearCount = &r
			}
		}
		b.providerSummary[k] = pSummary
	}
	return nil
}

func (b *categorySummaryBuilder) Build() []kafka.Doc {
	var docs []kafka.Doc
	for _, v := range b.connectionSummary {
		docs = append(docs, v)
		h := v
		h.ReportType = h.ReportType + "History"
		docs = append(docs, h)
	}
	for _, v := range b.providerSummary {
		docs = append(docs, v)
		h := v
		h.ReportType = h.ReportType + "History"
		docs = append(docs, h)
	}
	return docs
}

type ConnectionCategoryResourceCountQueryResponse struct {
	Hits ConnectionCategoryResourceCountQueryHits `json:"hits"`
}
type ConnectionCategoryResourceCountQueryHits struct {
	Total keibi.SearchTotal                         `json:"total"`
	Hits  []ConnectionCategoryResourceCountQueryHit `json:"hits"`
}
type ConnectionCategoryResourceCountQueryHit struct {
	ID      string                       `json:"_id"`
	Score   float64                      `json:"_score"`
	Index   string                       `json:"_index"`
	Type    string                       `json:"_type"`
	Version int64                        `json:"_version,omitempty"`
	Source  es.ConnectionCategorySummary `json:"_source"`
	Sort    []interface{}                `json:"sort"`
}

func (b *categorySummaryBuilder) queryCategoryConnectionResourceCount(scheduleJobID uint, connectionID string, category string) (int, error) {
	res := make(map[string]interface{})
	var filters []interface{}
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]interface{}{"report_type": {es.CategorySummary + "History"}},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]interface{}{"category_name": {category}},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]interface{}{"schedule_job_id": {scheduleJobID}},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]interface{}{"source_id": {connectionID}},
	})
	res["size"] = es.EsFetchPageSize
	res["query"] = map[string]interface{}{
		"bool": map[string]interface{}{
			"filter": filters,
		},
	}
	c, err := json.Marshal(res)
	if err != nil {
		return 0, err
	}

	var response ConnectionCategoryResourceCountQueryResponse
	err = b.client.Search(context.Background(), es.ConnectionSummaryIndex, string(c), &response)
	if err != nil {
		return 0, err
	}

	if len(response.Hits.Hits) == 0 {
		return 0, nil
	}
	return response.Hits.Hits[0].Source.ResourceCount, nil
}

type ProviderCategoryResourceCountQueryResponse struct {
	Hits ProviderCategoryResourceCountQueryHits `json:"hits"`
}
type ProviderCategoryResourceCountQueryHits struct {
	Total keibi.SearchTotal                       `json:"total"`
	Hits  []ProviderCategoryResourceCountQueryHit `json:"hits"`
}
type ProviderCategoryResourceCountQueryHit struct {
	ID      string                     `json:"_id"`
	Score   float64                    `json:"_score"`
	Index   string                     `json:"_index"`
	Type    string                     `json:"_type"`
	Version int64                      `json:"_version,omitempty"`
	Source  es.ProviderCategorySummary `json:"_source"`
	Sort    []interface{}              `json:"sort"`
}

func (b *categorySummaryBuilder) queryCategoryProviderResourceCount(scheduleJobID uint, provider source.Type, category string) (int, error) {
	res := make(map[string]interface{})
	var filters []interface{}
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]interface{}{"report_type": {es.CategoryProviderSummary + "History"}},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]interface{}{"category_name": {category}},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]interface{}{"schedule_job_id": {scheduleJobID}},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]interface{}{"source_type": {provider.String()}},
	})
	res["size"] = es.EsFetchPageSize
	res["query"] = map[string]interface{}{
		"bool": map[string]interface{}{
			"filter": filters,
		},
	}
	c, err := json.Marshal(res)
	if err != nil {
		return 0, err
	}

	var response ProviderCategoryResourceCountQueryResponse
	err = b.client.Search(context.Background(), es.ProviderSummaryIndex, string(c), &response)
	if err != nil {
		return 0, err
	}

	if len(response.Hits.Hits) == 0 {
		return 0, nil
	}
	return response.Hits.Hits[0].Source.ResourceCount, nil
}

func (b *categorySummaryBuilder) Cleanup(summarizeJobID uint) error {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must_not": []map[string]interface{}{
					{
						"term": map[string]interface{}{
							"summarize_job_id": summarizeJobID,
						},
					},
				},
				"filter": []map[string]interface{}{
					{
						"terms": map[string]interface{}{
							"report_type": []string{string(es.CategorySummary), string(es.CategoryProviderSummary)},
						},
					},
				},
			},
		},
	}

	esClient := b.client.ES()
	resp, err := keibi.DeleteByQuery(context.Background(), esClient, []string{es.ProviderSummaryIndex, es.ConnectionSummaryIndex}, query,
		esClient.DeleteByQuery.WithRefresh(true),
		esClient.DeleteByQuery.WithConflicts("proceed"),
	)
	if err != nil {
		return err
	}
	if len(resp.Failures) != 0 {
		body, err := json.Marshal(resp)
		if err != nil {
			return err
		}
		return fmt.Errorf("elasticsearch: delete by query: %s", string(body))
	}
	return nil
}
