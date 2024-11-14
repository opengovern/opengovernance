package spend

import (
	"github.com/opengovern/og-util/pkg/integration"
)

const (
	AnalyticsSpendIntegrationSummaryIndex = "analytics_spend_integration_summary"
)

type PerIntegrationMetricTrendSummary struct {
	DateEpoch       int64            `json:"date_epoch"`
	IntegrationID   string           `json:"integration_id"`
	IntegrationName string           `json:"integration_name"`
	IntegrationType integration.Type `json:"integration_type"`
	CostValue       float64          `json:"cost_value"`
	IsJobSuccessful bool             `json:"is_job_successful"`
}

type IntegrationMetricTrendSummary struct {
	EsID    string `json:"es_id"`
	EsIndex string `json:"es_index"`

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

	Integrations    []PerIntegrationMetricTrendSummary          `json:"integrations"`
	IntegrationsMap map[string]PerIntegrationMetricTrendSummary `json:"-"`
}

func (r IntegrationMetricTrendSummary) KeysAndIndex() ([]string, string) {
	keys := []string{
		r.Date,
		r.MetricID,
	}
	return keys, AnalyticsSpendIntegrationSummaryIndex
}
