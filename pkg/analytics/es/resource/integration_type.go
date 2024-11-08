package resource

import (
	"github.com/opengovern/og-util/pkg/integration"
)

const (
	AnalyticsIntegrationTypeSummaryIndex                    = "analytics_integration_type_summary"
	ResourceCollectionsAnalyticsIntegrationTypeSummaryIndex = "rc_analytics_integration_type_summary"
)

type PerIntegrationTypeMetricTrendSummary struct {
	IntegrationType                 integration.Type `json:"integration_type"`
	ResourceCount                   int              `json:"resource_count"`
	TotalIntegrationTypes           int64            `json:"total_integration_types"`
	TotalSuccessfulIntegrationTypes int64            `json:"total_successful_integration_types"`
}

type IntegrationTypeMetricTrendSummaryResult struct {
	TotalResourceCount int                                    `json:"total_resource_count"`
	IntegrationTypes   []PerIntegrationTypeMetricTrendSummary `json:"integration_types"`
}

type IntegrationTypeMetricTrendSummary struct {
	EsID    string `json:"es_id"`
	EsIndex string `json:"es_index"`

	EvaluatedAt int64  `json:"evaluated_at"`
	Date        string `json:"date"`
	Month       string `json:"month"`
	Year        string `json:"year"`
	MetricID    string `json:"metric_id"`
	MetricName  string `json:"metric_name"`

	IntegrationTypes    *IntegrationTypeMetricTrendSummaryResult           `json:"integration_types,omitempty"`
	ResourceCollections map[string]IntegrationTypeMetricTrendSummaryResult `json:"resource_collections,omitempty"`
}

func (r IntegrationTypeMetricTrendSummary) KeysAndIndex() ([]string, string) {
	keys := []string{
		r.Date,
		r.MetricID,
	}
	idx := AnalyticsIntegrationTypeSummaryIndex
	if r.ResourceCollections != nil {
		idx = ResourceCollectionsAnalyticsIntegrationTypeSummaryIndex
	}
	return keys, idx
}
