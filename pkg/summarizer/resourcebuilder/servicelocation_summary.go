package resourcebuilder

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-util/pkg/kafka"

	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
	"gitlab.com/keibiengine/keibi-engine/pkg/cloudservice"
	describe "gitlab.com/keibiengine/keibi-engine/pkg/describe/es"
	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"
)

type serviceLocationSummaryBuilder struct {
	client            keibi.Client
	summarizerJobID   uint
	connectionSummary map[string]es.ConnectionServiceLocationSummary
}

func NewServiceLocationSummaryBuilder(client keibi.Client, summarizerJobID uint) *serviceLocationSummaryBuilder {
	return &serviceLocationSummaryBuilder{
		client:            client,
		summarizerJobID:   summarizerJobID,
		connectionSummary: make(map[string]es.ConnectionServiceLocationSummary),
	}
}

func (b *serviceLocationSummaryBuilder) Process(resource describe.LookupResource) {
	key := resource.SourceID + "-" + cloudservice.ServiceNameByResourceType(resource.ResourceType)
	if _, ok := b.connectionSummary[key]; !ok {
		b.connectionSummary[key] = es.ConnectionServiceLocationSummary{
			SummarizeJobID:       b.summarizerJobID,
			ScheduleJobID:        resource.ScheduleJobID,
			SourceID:             resource.SourceID,
			SourceType:           resource.SourceType,
			SourceJobID:          resource.SourceJobID,
			ServiceName:          cloudservice.ServiceNameByResourceType(resource.ResourceType),
			LocationDistribution: map[string]int{},
			ReportType:           es.ServiceLocationConnectionSummary,
		}
	}

	v := b.connectionSummary[key]
	v.LocationDistribution[resource.Location]++
	b.connectionSummary[key] = v
}

func (b *serviceLocationSummaryBuilder) PopulateHistory(lastDayJobID, lastWeekJobID, lastQuarterJobID, lastYearJobID uint) error {
	return nil
}

func (b *serviceLocationSummaryBuilder) Build() []kafka.Doc {
	var docs []kafka.Doc
	for _, v := range b.connectionSummary {
		docs = append(docs, v)
	}
	return docs
}

func (b *serviceLocationSummaryBuilder) Cleanup(summarizeJobID uint) error {
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
							"report_type": []string{string(es.ServiceLocationConnectionSummary)},
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
