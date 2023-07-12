package resourcebuilder

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kaytu-io/kaytu-util/pkg/kafka"

	describe "github.com/kaytu-io/kaytu-engine/pkg/describe/es"
	"github.com/kaytu-io/kaytu-engine/pkg/inventory"
	"github.com/kaytu-io/kaytu-engine/pkg/summarizer/es"
	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
	"go.uber.org/zap"
)

type resourceTypeSummaryBuilder struct {
	client            keibi.Client
	logger            *zap.Logger
	db                inventory.Database
	summarizerJobID   uint
	connectionSummary map[string]es.ConnectionResourceTypeSummary
	providerSummary   map[string]es.ProviderResourceTypeSummary
}

func NewResourceTypeSummaryBuilder(client keibi.Client, logger *zap.Logger, db inventory.Database, summarizerJobID uint) *resourceTypeSummaryBuilder {
	return &resourceTypeSummaryBuilder{
		client:            client,
		db:                db,
		logger:            logger,
		summarizerJobID:   summarizerJobID,
		connectionSummary: make(map[string]es.ConnectionResourceTypeSummary),
		providerSummary:   make(map[string]es.ProviderResourceTypeSummary),
	}
}

func (b *resourceTypeSummaryBuilder) Process(resource describe.LookupResource) {
	key := string(resource.SourceID) + "---" + resource.ResourceType
	if _, ok := b.connectionSummary[key]; !ok {
		b.connectionSummary[key] = es.ConnectionResourceTypeSummary{
			SummarizeJobID:   b.summarizerJobID,
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
			SummarizeJobID:   b.summarizerJobID,
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

func (b *resourceTypeSummaryBuilder) Build() []kafka.Doc {
	var docs []kafka.Doc
	for _, v := range b.connectionSummary {
		docs = append(docs, v)
		if err := b.db.CreateOrUpdateMetric(inventory.Metric{
			SourceID:         v.SourceID,
			Provider:         v.SourceType.String(),
			ResourceType:     v.ResourceType,
			ScheduleJobID:    v.ScheduleJobID,
			SummarizeJobID:   &v.SummarizeJobID,
			LastDayCount:     v.LastDayCount,
			LastWeekCount:    v.LastWeekCount,
			LastQuarterCount: v.LastQuarterCount,
			LastYearCount:    v.LastYearCount,
			Count:            v.ResourceCount,
		}); err != nil {
			b.logger.Error("failed to create metrics due to error", zap.Error(err))
		}

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

func (b *resourceTypeSummaryBuilder) Cleanup(summarizeJobID uint) error {
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
							"report_type": []string{string(es.ResourceTypeSummary), string(es.ResourceTypeProviderSummary)},
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
