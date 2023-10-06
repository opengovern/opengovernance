package resource

import (
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"strconv"
)

const (
	AnalyticsConnectorSummaryIndex                    = "analytics_connector_summary"
	ResourceCollectionsAnalyticsConnectorSummaryIndex = "rc_analytics_connector_summary"
)

type ConnectorMetricTrendSummary struct {
	Connector                  source.Type `json:"connector"`
	EvaluatedAt                int64       `json:"evaluated_at"`
	Date                       string      `json:"date"`
	Month                      string      `json:"month"`
	Year                       string      `json:"year"`
	MetricID                   string      `json:"metric_id"`
	MetricName                 string      `json:"metric_name"`
	ResourceCount              int         `json:"resource_count"`
	TotalConnections           int64       `json:"total_connections"`
	TotalSuccessfulConnections int64       `json:"total_successful_connections"`

	ResourceCollection *string `json:"resource_collection"`
}

func (r ConnectorMetricTrendSummary) KeysAndIndex() ([]string, string) {
	keys := []string{
		r.Connector.String(),
		r.MetricID,
		strconv.FormatInt(r.EvaluatedAt, 10),
	}
	idx := AnalyticsConnectorSummaryIndex
	if r.ResourceCollection != nil {
		keys = append(keys, *r.ResourceCollection)
		idx = ResourceCollectionsAnalyticsConnectorSummaryIndex
	}
	return keys, idx
}
