package resource

import (
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

const (
	AnalyticsConnectionSummaryIndex                    = "analytics_connection_summary"
	ResourceCollectionsAnalyticsConnectionSummaryIndex = "rc_analytics_connection_summary"
)

type PerConnectionMetricTrendSummary struct {
	Connector       source.Type `json:"connector"`
	ConnectionID    string      `json:"connection_id"`
	ConnectionName  string      `json:"connection_name"`
	ResourceCount   int         `json:"resource_count"`
	IsJobSuccessful bool        `json:"is_job_successful"`
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

	Connections         *ConnectionMetricTrendSummaryResult           `json:"connections,omitempty"`
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
