package compliancebuilder

import (
	"context"
	"encoding/json"
	"fmt"

	es2 "gitlab.com/keibiengine/keibi-engine/pkg/compliance/es"

	"gitlab.com/keibiengine/keibi-engine/pkg/kafka"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"
)

type benchmarkSummaryBuilder struct {
	client           keibi.Client
	summarizerJobID  uint
	benchmarkSummary map[string]es.BenchmarkSummary
}

func NewBenchmarkSummaryBuilder(client keibi.Client, summarizerJobID uint) *benchmarkSummaryBuilder {
	return &benchmarkSummaryBuilder{
		client:           client,
		summarizerJobID:  summarizerJobID,
		benchmarkSummary: make(map[string]es.BenchmarkSummary),
	}
}

func (b *benchmarkSummaryBuilder) Process(resource es2.Finding) error {
	if _, ok := b.benchmarkSummary[resource.BenchmarkID]; !ok {
		b.benchmarkSummary[resource.BenchmarkID] = es.BenchmarkSummary{
			BenchmarkID:   resource.BenchmarkID,
			ScheduleJobID: resource.ScheduleJobID,
			DescribedAt:   resource.DescribedAt,
			EvaluatedAt:   resource.EvaluatedAt,
			Policies:      nil,
			ReportType:    es.BenchmarksSummary,
		}
	}

	resourceResult := es.ResourceResult{
		ResourceID:   resource.ResourceID,
		ResourceName: resource.ResourceName,
		SourceID:     resource.ConnectionID,
		Result:       resource.Result,
	}

	v := b.benchmarkSummary[resource.BenchmarkID]
	found := false
	for idx, p := range v.Policies {
		if p.PolicyID == resource.PolicyID {
			v.Policies[idx].Resources = append(v.Policies[idx].Resources, resourceResult)
			found = true
		}
	}
	if !found {
		v.Policies = append(v.Policies, es.PolicySummary{
			PolicyID:  resource.PolicyID,
			Resources: []es.ResourceResult{resourceResult},
		})
	}
	b.benchmarkSummary[resource.BenchmarkID] = v
	return nil
}

func (b *benchmarkSummaryBuilder) PopulateHistory(lastDayJobID, lastWeekJobID, lastQuarterJobID, lastYearJobID uint) error {
	return nil
}

func (b *benchmarkSummaryBuilder) Build() []kafka.Doc {
	var docs []kafka.Doc
	for _, v := range b.benchmarkSummary {
		docs = append(docs, v)
		//h := v
		//h.ReportType = h.ReportType + "History"
		//docs = append(docs, h)
	}
	return docs
}

func (b *benchmarkSummaryBuilder) Cleanup(scheduleJobID uint) error {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must_not": []map[string]interface{}{
					{
						"term": map[string]interface{}{
							"schedule_job_id": scheduleJobID,
						},
					},
				},
				"filter": []map[string]interface{}{
					{
						"terms": map[string]interface{}{
							"report_type": []string{string(es.BenchmarksSummary)},
						},
					},
				},
			},
		},
	}

	esClient := b.client.ES()
	resp, err := keibi.DeleteByQuery(context.Background(), esClient, []string{es.BenchmarkSummaryIndex}, query,
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
