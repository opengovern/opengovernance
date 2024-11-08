package resource

import (
	"github.com/opengovern/og-util/pkg/integration"
)

const (
	AnalyticsConnectorSummaryIndex                    = "analytics_connector_summary"
	ResourceCollectionsAnalyticsConnectorSummaryIndex = "rc_analytics_connector_summary"
)

type PerConnectorMetricTrendSummary struct {
	IntegrationType            integration.Type `json:"integration_type"`
	ResourceCount              int              `json:"resource_count"`
	TotalConnections           int64            `json:"total_connections"`
	TotalSuccessfulConnections int64            `json:"total_successful_connections"`
}

type ConnectorMetricTrendSummaryResult struct {
	TotalResourceCount int                              `json:"total_resource_count"`
	IntegrationTypes   []PerConnectorMetricTrendSummary `json:"integration_types"`
}

type ConnectorMetricTrendSummary struct {
	EsID    string `json:"es_id"`
	EsIndex string `json:"es_index"`

	EvaluatedAt int64  `json:"evaluated_at"`
	Date        string `json:"date"`
	Month       string `json:"month"`
	Year        string `json:"year"`
	MetricID    string `json:"metric_id"`
	MetricName  string `json:"metric_name"`

	IntegrationTypes    *ConnectorMetricTrendSummaryResult           `json:"integration_types,omitempty"`
	ResourceCollections map[string]ConnectorMetricTrendSummaryResult `json:"resource_collections,omitempty"`
}

func (r ConnectorMetricTrendSummary) KeysAndIndex() ([]string, string) {
	keys := []string{
		r.Date,
		r.MetricID,
	}
	idx := AnalyticsConnectorSummaryIndex
	if r.ResourceCollections != nil {
		idx = ResourceCollectionsAnalyticsConnectorSummaryIndex
	}
	return keys, idx
}
