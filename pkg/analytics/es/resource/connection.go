package resource

import (
	"github.com/opengovern/og-util/pkg/integration"
)

const (
	AnalyticsConnectionSummaryIndex                    = "analytics_connection_summary"
	ResourceCollectionsAnalyticsConnectionSummaryIndex = "rc_analytics_connection_summary"
)

type PerConnectionMetricTrendSummary struct {
	IntegrationType integration.Type `json:"integration_type"`
	IntegrationID   string           `json:"integration_id"`
	IntegrationName string           `json:"integration_name"`
	ResourceCount   int              `json:"resource_count"`
	IsJobSuccessful bool             `json:"is_job_successful"`
}

type ConnectionMetricTrendSummaryResult struct {
	TotalResourceCount int                               `json:"total_resource_count"`
	Connections        []PerConnectionMetricTrendSummary `json:"connections"`
}

type ConnectionMetricTrendSummary struct {
	EsID    string `json:"es_id"`
	EsIndex string `json:"es_index"`

	EvaluatedAt int64  `json:"evaluated_at"`
	Date        string `json:"date"`
	Month       string `json:"month"`
	Year        string `json:"year"`
	MetricID    string `json:"metric_id"`
	MetricName  string `json:"metric_name"`

	Integrations        *ConnectionMetricTrendSummaryResult           `json:"integrations,omitempty"`
	ResourceCollections map[string]ConnectionMetricTrendSummaryResult `json:"resource_collections,omitempty"`
}

func (r ConnectionMetricTrendSummary) KeysAndIndex() ([]string, string) {
	keys := []string{
		r.Date,
		r.MetricID,
	}
	idx := AnalyticsConnectionSummaryIndex
	if r.ResourceCollections != nil {
		idx = ResourceCollectionsAnalyticsConnectionSummaryIndex
	}
	return keys, idx
}
