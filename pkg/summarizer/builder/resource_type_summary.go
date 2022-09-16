package builder

import (
	"context"
	"encoding/json"
	"fmt"

	describe "gitlab.com/keibiengine/keibi-engine/pkg/describe/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/kafka"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
	"gitlab.com/keibiengine/keibi-engine/pkg/source"
	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"
)

type resourceTypeSummaryBuilder struct {
	client            keibi.Client
	summarizerJobID   uint
	connectionSummary map[string]es.ConnectionResourceTypeSummary
	providerSummary   map[string]es.ProviderResourceTypeSummary
}

func NewResourceTypeSummaryBuilder(client keibi.Client, summarizerJobID uint) *resourceTypeSummaryBuilder {
	return &resourceTypeSummaryBuilder{
		client:            client,
		summarizerJobID:   summarizerJobID,
		connectionSummary: make(map[string]es.ConnectionResourceTypeSummary),
		providerSummary:   make(map[string]es.ProviderResourceTypeSummary),
	}
}

func (b *resourceTypeSummaryBuilder) Process(resource describe.LookupResource) {
	key := string(resource.SourceID) + "---" + resource.ResourceType
	if _, ok := b.connectionSummary[key]; !ok {
		b.connectionSummary[key] = es.ConnectionResourceTypeSummary{
			ScheduleJobID:    resource.ScheduleJobID,
			SourceID:         resource.SourceID,
			SourceJobID:      resource.SourceJobID,
			ResourceType:     resource.ResourceType,
			SourceType:       resource.SourceType,
			ResourceCount:    0,
			LastDayCount:     nil,
			LastWeekCount:    nil,
			LastQuarterCount: nil,
			LastYearCount:    nil,
			ReportType:       es.ResourceTypeSummary,
		}
	}

	v := b.connectionSummary[key]
	v.ResourceCount++
	b.connectionSummary[key] = v

	key = string(resource.SourceType) + "---" + resource.ResourceType
	if _, ok := b.providerSummary[key]; !ok {
		b.providerSummary[key] = es.ProviderResourceTypeSummary{
			ScheduleJobID:    resource.ScheduleJobID,
			ResourceType:     resource.ResourceType,
			SourceType:       resource.SourceType,
			ResourceCount:    0,
			LastDayCount:     nil,
			LastWeekCount:    nil,
			LastQuarterCount: nil,
			LastYearCount:    nil,
			ReportType:       es.ResourceTypeProviderSummary,
		}
	}

	v2 := b.providerSummary[key]
	v2.ResourceCount++
	b.providerSummary[key] = v2
}

func (b *resourceTypeSummaryBuilder) PopulateHistory(lastDayJobID, lastWeekJobID, lastQuarterJobID, lastYearJobID uint) error {
	jobIDs := []uint{lastDayJobID, lastWeekJobID, lastQuarterJobID, lastYearJobID}
	for k, connSummary := range b.connectionSummary {
		for idx, jid := range jobIDs {
			if jid == 0 {
				continue
			}

			r, err := b.queryResourceTypeConnectionResourceCount(jid, connSummary.SourceID, connSummary.ResourceType)
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

			r, err := b.queryResourceTypeProviderResourceCount(jid, pSummary.SourceType, pSummary.ResourceType)
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

func (b *resourceTypeSummaryBuilder) Build() []kafka.Doc {
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

type ConnectionResourceTypeResourceCountQueryResponse struct {
	Hits ConnectionResourceTypeResourceCountQueryHits `json:"hits"`
}
type ConnectionResourceTypeResourceCountQueryHits struct {
	Total keibi.SearchTotal                             `json:"total"`
	Hits  []ConnectionResourceTypeResourceCountQueryHit `json:"hits"`
}
type ConnectionResourceTypeResourceCountQueryHit struct {
	ID      string                           `json:"_id"`
	Score   float64                          `json:"_score"`
	Index   string                           `json:"_index"`
	Type    string                           `json:"_type"`
	Version int64                            `json:"_version,omitempty"`
	Source  es.ConnectionResourceTypeSummary `json:"_source"`
	Sort    []interface{}                    `json:"sort"`
}

func (b *resourceTypeSummaryBuilder) queryResourceTypeConnectionResourceCount(scheduleJobID uint, connectionID string, resourceType string) (int, error) {
	res := make(map[string]interface{})
	var filters []interface{}
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]interface{}{"report_type": {es.ResourceTypeSummary + "History"}},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]interface{}{"resource_type": {resourceType}},
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

	var response ConnectionResourceTypeResourceCountQueryResponse
	err = b.client.Search(context.Background(), es.ConnectionSummaryIndex, string(c), &response)
	if err != nil {
		return 0, err
	}

	if len(response.Hits.Hits) == 0 {
		return 0, nil
	}
	return response.Hits.Hits[0].Source.ResourceCount, nil
}

type ProviderResourceTypeResourceCountQueryResponse struct {
	Hits ProviderResourceTypeResourceCountQueryHits `json:"hits"`
}
type ProviderResourceTypeResourceCountQueryHits struct {
	Total keibi.SearchTotal                           `json:"total"`
	Hits  []ProviderResourceTypeResourceCountQueryHit `json:"hits"`
}
type ProviderResourceTypeResourceCountQueryHit struct {
	ID      string                     `json:"_id"`
	Score   float64                    `json:"_score"`
	Index   string                     `json:"_index"`
	Type    string                     `json:"_type"`
	Version int64                      `json:"_version,omitempty"`
	Source  es.ProviderCategorySummary `json:"_source"`
	Sort    []interface{}              `json:"sort"`
}

func (b *resourceTypeSummaryBuilder) queryResourceTypeProviderResourceCount(scheduleJobID uint, provider source.Type, resourceType string) (int, error) {
	res := make(map[string]interface{})
	var filters []interface{}
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]interface{}{"report_type": {es.ResourceTypeProviderSummary + "History"}},
	})
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]interface{}{"resource_type": {resourceType}},
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

	var response ProviderResourceTypeResourceCountQueryResponse
	err = b.client.Search(context.Background(), es.ProviderSummaryIndex, string(c), &response)
	if err != nil {
		return 0, err
	}

	if len(response.Hits.Hits) == 0 {
		return 0, nil
	}
	return response.Hits.Hits[0].Source.ResourceCount, nil
}

func (b *resourceTypeSummaryBuilder) Cleanup(scheduleJobID uint) error {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": []map[string]interface{}{
					{
						"terms": map[string]interface{}{
							"report_type": []string{string(es.ResourceTypeSummary), string(es.ResourceTypeProviderSummary)},
						},
					},
					{
						"bool": map[string]interface{}{
							"must_not": map[string]interface{}{
								"schedule_job_id": scheduleJobID,
							},
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
