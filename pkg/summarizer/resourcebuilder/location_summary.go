package resourcebuilder

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kaytu-io/kaytu-util/pkg/kafka"

	describe "github.com/kaytu-io/kaytu-engine/pkg/describe/es"
	"github.com/kaytu-io/kaytu-engine/pkg/summarizer/es"
	"github.com/kaytu-io/kaytu-util/pkg/keibi-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type locationSummaryBuilder struct {
	client            keibi.Client
	summarizerJobID   uint
	connectionSummary map[string]es.ConnectionLocationSummary
	providerSummary   map[source.Type]es.ProviderLocationSummary
}

func NewLocationSummaryBuilder(client keibi.Client, summarizerJobID uint) *locationSummaryBuilder {
	return &locationSummaryBuilder{
		client:            client,
		summarizerJobID:   summarizerJobID,
		connectionSummary: make(map[string]es.ConnectionLocationSummary),
		providerSummary:   make(map[source.Type]es.ProviderLocationSummary),
	}
}

func (b *locationSummaryBuilder) Process(resource describe.LookupResource) {
	if _, ok := b.connectionSummary[resource.SourceID]; !ok {
		b.connectionSummary[resource.SourceID] = es.ConnectionLocationSummary{
			SummarizeJobID:       b.summarizerJobID,
			SummarizedAt:         time.Now().Unix(),
			SourceID:             resource.SourceID,
			SourceType:           resource.SourceType,
			SourceJobID:          resource.SourceJobID,
			LocationDistribution: map[string]int{},
			ReportType:           es.LocationConnectionSummary,
		}
	}

	if _, ok := b.providerSummary[resource.SourceType]; !ok {
		b.providerSummary[resource.SourceType] = es.ProviderLocationSummary{
			SummarizeJobID:       b.summarizerJobID,
			SummarizedAt:         time.Now().Unix(),
			SourceType:           resource.SourceType,
			LocationDistribution: map[string]int{},
			ReportType:           es.LocationProviderSummary,
		}
	}

	if resource.Location == "" {
		fmt.Printf("resource from account %s of type %s from connector %s has no location ignoring it in location summary\n", resource.SourceID, resource.ResourceType, resource.SourceType)
		return
	}
	v := b.connectionSummary[resource.SourceID]
	v.LocationDistribution[resource.Location]++
	b.connectionSummary[resource.SourceID] = v
	v2 := b.providerSummary[resource.SourceType]
	v2.LocationDistribution[resource.Location]++
	b.providerSummary[resource.SourceType] = v2
}

func (b *locationSummaryBuilder) Build() []kafka.Doc {
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

func (b *locationSummaryBuilder) Cleanup(summarizeJobID uint) error {
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
							"report_type": []string{string(es.LocationConnectionSummary), string(es.LocationProviderSummary)},
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
