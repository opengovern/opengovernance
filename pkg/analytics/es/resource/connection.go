package resource

import (
	"github.com/opengovern/og-util/pkg/integration"
)

const (
	AnalyticsIntegrationSummaryIndex                    = "analytics_integration_summary"
	ResourceCollectionsAnalyticsIntegrationSummaryIndex = "rc_analytics_integration_summary"
)

type PerIntegrationMetricTrendSummary struct {
	IntegrationType integration.Type `json:"integration_type"`
	IntegrationID   string           `json:"integration_id"`
	IntegrationName string           `json:"integration_name"`
	ResourceCount   int              `json:"resource_count"`
	IsJobSuccessful bool             `json:"is_job_successful"`
}

type IntegrationMetricTrendSummaryResult struct {
	TotalResourceCount int                                `json:"total_resource_count"`
	Integrations       []PerIntegrationMetricTrendSummary `json:"integrations"`
}

type IntegrationMetricTrendSummary struct {
	EsID    string `json:"es_id"`
	EsIndex string `json:"es_index"`

	EvaluatedAt int64  `json:"evaluated_at"`
	Date        string `json:"date"`
	Month       string `json:"month"`
	Year        string `json:"year"`
	MetricID    string `json:"metric_id"`
	MetricName  string `json:"metric_name"`

	Integrations        *IntegrationMetricTrendSummaryResult           `json:"integrations,omitempty"`
	ResourceCollections map[string]IntegrationMetricTrendSummaryResult `json:"resource_collections,omitempty"`
}

func (r IntegrationMetricTrendSummary) KeysAndIndex() ([]string, string) {
	keys := []string{
		r.Date,
		r.MetricID,
	}
	idx := AnalyticsIntegrationSummaryIndex
	if r.ResourceCollections != nil {
		idx = ResourceCollectionsAnalyticsIntegrationSummaryIndex
	}
	return keys, idx
}
