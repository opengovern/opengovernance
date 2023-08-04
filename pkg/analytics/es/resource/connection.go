package resource

import (
	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"strconv"
)

const (
	AnalyticsConnectionSummaryIndex = "analytics_connection_summary"
)

type ConnectionMetricTrendSummary struct {
	ConnectionID  uuid.UUID   `json:"connection_id"`
	Connector     source.Type `json:"connector"`
	EvaluatedAt   int64       `json:"evaluated_at"`
	MetricID      string      `json:"metric_id"`
	ResourceCount int         `json:"resource_count"`
}

func (r ConnectionMetricTrendSummary) KeysAndIndex() ([]string, string) {
	keys := []string{
		strconv.FormatInt(r.EvaluatedAt, 10),
		r.ConnectionID.String(),
		r.MetricID,
	}
	return keys, AnalyticsConnectionSummaryIndex
}
