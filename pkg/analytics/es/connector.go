package es

import (
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"strconv"
)

const (
	AnalyticsConnectorSummaryIndex = "analytics_connector_summary"
)

type ConnectorMetricTrendSummary struct {
	Connector     source.Type `json:"connector"`
	EvaluatedAt   int64       `json:"evaluated_at"`
	MetricID      string      `json:"metric_id"`
	ResourceCount int         `json:"resource_count"`
}

func (r ConnectorMetricTrendSummary) KeysAndIndex() ([]string, string) {
	keys := []string{
		r.Connector.String(),
		r.MetricID,
		strconv.FormatInt(r.EvaluatedAt, 10),
	}
	return keys, AnalyticsConnectorSummaryIndex
}
