package resourcebuilder

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kaytu-io/kaytu-util/pkg/kafka"

	describe "github.com/kaytu-io/kaytu-engine/pkg/describe/es"
	"github.com/kaytu-io/kaytu-engine/pkg/summarizer/es"
	"github.com/kaytu-io/kaytu-util/pkg/kaytu-es-sdk"
)

type resourceSummaryBuilder struct {
	client            kaytu.Client
	summarizerJobID   uint
	connectionSummary map[string]es.ConnectionResourcesSummary
}

func NewResourceSummaryBuilder(client kaytu.Client, summarizerJobID uint) *resourceSummaryBuilder {
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
	resp, err := kaytu.DeleteByQuery(context.Background(), esClient, []string{es.ProviderSummaryIndex, es.ConnectionSummaryIndex}, query,
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
