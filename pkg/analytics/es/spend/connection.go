package spend

import (
	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

const (
	AnalyticsSpendConnectionSummaryIndex = "analytics_spend_connection_summary"
)

type PerConnectionMetricTrendSummary struct {
	DateEpoch       int64       `json:"date_epoch"`
	ConnectionID    string      `json:"connection_id"`
	ConnectionName  string      `json:"connection_name"`
	Connector       source.Type `json:"connector"`
	CostValue       float64     `json:"cost_value"`
	IsJobSuccessful bool        `json:"is_job_successful"`
}

type ConnectionMetricTrendSummary struct {
	MetricName     string  `json:"metric_name"`
	MetricID       string  `json:"metric_id"`
	TotalCostValue float64 `json:"total_cost_value"`

	EvaluatedAt int64  `json:"evaluated_at"`
	Date        string `json:"date"`
	DateEpoch   int64  `json:"date_epoch"`
	Month       string `json:"month"`
	Year        string `json:"year"`
	PeriodStart int64  `json:"period_start"`
	PeriodEnd   int64  `json:"period_end"`

	Connections    []PerConnectionMetricTrendSummary          `json:"connections"`
	ConnectionsMap map[string]PerConnectionMetricTrendSummary `json:"-"`

	// Deprecated
	ConnectionID uuid.UUID `json:"connection_id"`
	// Deprecated
	ConnectionName string `json:"connection_name"`
	// Deprecated
	Connector source.Type `json:"connector"`
	// Deprecated
	CostValue float64 `json:"cost_value"`
	// Deprecated
	IsJobSuccessful bool `json:"is_job_successful"`
}

func (r ConnectionMetricTrendSummary) KeysAndIndex() ([]string, string) {
	keys := []string{
		r.Date,
		r.MetricID,
	}
	return keys, AnalyticsSpendConnectionSummaryIndex
}
