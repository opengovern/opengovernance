package spend

import (
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

const (
	AnalyticsSpendConnectorSummaryIndex = "analytics_spend_connector_summary"
)

type ConnectorMetricTrendSummary struct {
	Connector                  source.Type `json:"connector"`
	Date                       string      `json:"date"`
	DateEpoch                  int64       `json:"date_epoch"`
	Month                      string      `json:"month"`
	Year                       string      `json:"year"`
	MetricID                   string      `json:"metric_id"`
	MetricName                 string      `json:"metric_name"`
	CostValue                  float64     `json:"cost_value"`
	PeriodStart                int64       `json:"period_start"`
	PeriodEnd                  int64       `json:"period_end"`
	TotalConnections           int64       `json:"total_connections"`
	TotalSuccessfulConnections int64       `json:"total_successful_connections"`
}

func (r ConnectorMetricTrendSummary) KeysAndIndex() ([]string, string) {
	keys := []string{
		r.Date,
		r.Connector.String(),
		r.MetricID,
	}
	return keys, AnalyticsSpendConnectorSummaryIndex
}
