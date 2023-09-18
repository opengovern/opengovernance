package spend

import (
	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

const (
	AnalyticsSpendConnectionSummaryIndex = "analytics_spend_connection_summary"
)

type ConnectionMetricTrendSummary struct {
	ConnectionID    uuid.UUID   `json:"connection_id"`
	ConnectionName  string      `json:"connection_name"`
	Connector       source.Type `json:"connector"`
	Date            string      `json:"date"`
	DateEpoch       int64       `json:"date_epoch"`
	Month           string      `json:"month"`
	Year            string      `json:"year"`
	MetricID        string      `json:"metric_id"`
	MetricName      string      `json:"metric_name"`
	CostValue       float64     `json:"cost_value"`
	PeriodStart     int64       `json:"period_start"`
	PeriodEnd       int64       `json:"period_end"`
	IsJobSuccessful bool        `json:"is_job_successful"`
}

func (r ConnectionMetricTrendSummary) KeysAndIndex() ([]string, string) {
	keys := []string{
		r.Date,
		r.ConnectionID.String(),
		r.MetricID,
	}
	return keys, AnalyticsSpendConnectionSummaryIndex
}
