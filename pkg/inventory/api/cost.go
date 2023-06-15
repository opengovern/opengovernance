package api

import (
	"time"

	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type CostTrendDatapoint struct {
	Cost float64   `json:"count"`
	Date time.Time `json:"date"`
}

type CostMetric struct {
	Connector            source.Type `json:"connector"`
	CostDimensionName    string      `json:"cost_dimension_name"`
	TotalCost            *float64    `json:"total_cost,omitempty"`
	DailyCostAtStartTime *float64    `json:"daily_cost_at_start_time,omitempty"`
	DailyCostAtEndTime   *float64    `json:"daily_cost_at_end_time,omitempty"`
}

type ListCostMetricsResponse struct {
	TotalCount int          `json:"total_count"`
	TotalCost  float64      `json:"total_cost"`
	Metrics    []CostMetric `json:"metrics"`
}

type ListCostCompositionResponse struct {
	TotalCount     int                `json:"total_count"`
	TotalCostValue float64            `json:"total_cost_value"`
	TopValues      map[string]float64 `json:"top_values"`
	Others         float64            `json:"others"`
}
