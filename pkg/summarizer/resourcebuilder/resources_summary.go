package resourcebuilder

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"

	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
	describe "gitlab.com/keibiengine/keibi-engine/pkg/describe/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"
)

type resourceSummaryBuilder struct {
	client            keibi.Client
	summarizerJobID   uint
	connectionSummary map[string]es.ConnectionResourcesSummary
}

func NewResourceSummaryBuilder(client keibi.Client, summarizerJobID uint) *resourceSummaryBuilder {
	return &resourceSummaryBuilder{
		client:            client,
		summarizerJobID:   summarizerJobID,
		connectionSummary: make(map[string]es.ConnectionResourcesSummary),
	}
}

func (b *resourceSummaryBuilder) Process(resource describe.LookupResource) {
	if _, ok := b.connectionSummary[resource.SourceID]; !ok {
		b.connectionSummary[resource.SourceID] = es.ConnectionResourcesSummary{
			SummarizeJobID:   b.summarizerJobID,
			ScheduleJobID:    resource.ScheduleJobID,
			SourceID:         resource.SourceID,
			SourceType:       resource.SourceType,
			SourceJobID:      resource.SourceJobID,
			DescribedAt:      resource.CreatedAt,
			ResourceCount:    0,
			LastDayCount:     nil,
			LastWeekCount:    nil,
			LastQuarterCount: nil,
			LastYearCount:    nil,
			ReportType:       es.ResourceSummary,
		}
	}

	v := b.connectionSummary[resource.SourceID]
	v.ResourceCount++
	b.connectionSummary[resource.SourceID] = v
}

func (b *resourceSummaryBuilder) PopulateHistory(lastDayJobID, lastWeekJobID, lastQuarterJobID, lastYearJobID uint) error {
	jobIDs := []uint{lastDayJobID, lastWeekJobID, lastQuarterJobID, lastYearJobID}
	for connId, connSummary := range b.connectionSummary {
		for idx, jid := range jobIDs {
			if jid == 0 {
				continue
			}

			r, err := b.queryConnectionResourceCount(jid, connId)
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
	}
	return nil
}

func (b *resourceSummaryBuilder) Build() []kafka.Doc {
	var docs []kafka.Doc
	for _, v := range b.connectionSummary {
		docs = append(docs, v)
		h := v
		h.ReportType = h.ReportType + "History"
		docs = append(docs, h)
	}
	return docs
}

type ConnectionResourceCountQueryResponse struct {
	Hits ConnectionResourceCountQueryHits `json:"hits"`
}
type ConnectionResourceCountQueryHits struct {
	Total keibi.SearchTotal                 `json:"total"`
	Hits  []ConnectionResourceCountQueryHit `json:"hits"`
}
type ConnectionResourceCountQueryHit struct {
	ID      string                        `json:"_id"`
	Score   float64                       `json:"_score"`
	Index   string                        `json:"_index"`
	Type    string                        `json:"_type"`
	Version int64                         `json:"_version,omitempty"`
	Source  es.ConnectionResourcesSummary `json:"_source"`
	Sort    []interface{}                 `json:"sort"`
}

func (b *resourceSummaryBuilder) queryConnectionResourceCount(scheduleJobID uint, connectionID string) (int, error) {
	res := make(map[string]interface{})
	var filters []interface{}
	filters = append(filters, map[string]interface{}{
		"terms": map[string][]interface{}{"report_type": {es.ResourceSummary + "History"}},
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

	var response ConnectionResourceCountQueryResponse
	err = b.client.Search(context.Background(), es.ConnectionSummaryIndex, string(c), &response)
	if err != nil {
		return 0, err
	}

	if len(response.Hits.Hits) == 0 {
		return 0, nil
	}
	return response.Hits.Hits[0].Source.ResourceCount, nil
}

func (b *resourceSummaryBuilder) Cleanup(summarizeJobID uint) error {
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
							"report_type": []string{string(es.ResourceSummary)},
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
