package compliancebuilder

import (
	"context"
	"encoding/json"
	"fmt"

	"gitlab.com/keibiengine/keibi-engine/pkg/types"

	es2 "gitlab.com/keibiengine/keibi-engine/pkg/compliance/es"

	"gitlab.com/keibiengine/keibi-engine/pkg/kafka"
	"gitlab.com/keibiengine/keibi-engine/pkg/keibi-es-sdk"
	"gitlab.com/keibiengine/keibi-engine/pkg/summarizer/es"
)

type findingMetricsBuilder struct {
	client          keibi.Client
	summarizerJobID uint
	metrics         es.FindingMetrics
}

func NewMetricsBuilder(client keibi.Client, summarizerJobID uint) *findingMetricsBuilder {
	return &findingMetricsBuilder{
		client:          client,
		summarizerJobID: summarizerJobID,
		metrics:         es.FindingMetrics{},
	}
}

func (b *findingMetricsBuilder) Process(resource es2.Finding) error {
	b.metrics.ScheduleJobID = b.summarizerJobID
	b.metrics.EvaluatedAt = resource.EvaluatedAt
	b.metrics.DescribedAt = resource.DescribedAt
	switch resource.Status {
	case types.ComplianceResultOK:
		b.metrics.PassedFindingsCount++
	case types.ComplianceResultALARM, types.ComplianceResultERROR:
		b.metrics.FailedFindingsCount++
	case types.ComplianceResultINFO, types.ComplianceResultSKIP:
		b.metrics.UnknownFindingsCount++
	}
	return nil
}

func (b *findingMetricsBuilder) PopulateHistory(lastDayJobID, lastWeekJobID, lastQuarterJobID, lastYearJobID uint) error {
	return nil
}

func (b *findingMetricsBuilder) Build() []kafka.Doc {
	return []kafka.Doc{b.metrics}
}

func (b *findingMetricsBuilder) Cleanup(scheduleJobID uint) error {
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
			},
		},
	}

	esClient := b.client.ES()
	resp, err := keibi.DeleteByQuery(context.Background(), esClient, []string{es.MetricsIndex}, query,
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
