package spend

import (
	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

const (
	AnalyticsSpendConnectionSummaryIndex = "analytics_spend_connection_summary"
)

type ConnectionMetricTrendSummary struct {
	ConnectionID uuid.UUID   `json:"connection_id"`
	Connector    source.Type `json:"connector"`
	Date         string      `json:"date"`
	MetricID     string      `json:"metric_id"`
	CostValue    float64     `json:"cost_value"`
}

func (r ConnectionMetricTrendSummary) KeysAndIndex() ([]string, string) {
	keys := []string{
		r.Date,
		r.ConnectionID.String(),
		r.MetricID,
	}
	return keys, AnalyticsSpendConnectionSummaryIndex
}
