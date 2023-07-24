package api

import (
	"time"

	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type CostTrendDatapoint struct {
	Cost float64   `json:"count"`
	Date time.Time `json:"date" format:"date"`
}

type ListServicesCostTrendDatapoint struct {
	ServiceName string               `json:"serviceName" example:""`
	CostTrend   []CostTrendDatapoint `json:"costTrend"`
}

type CostMetric struct {
	Connector            source.Type `json:"connector" example:"azure"`
	CostDimensionName    string      `json:"cost_dimension_name" `
	TotalCost            *float64    `json:"total_cost,omitempty"`
	DailyCostAtStartTime *float64    `json:"daily_cost_at_start_time,omitempty"`
	DailyCostAtEndTime   *float64    `json:"daily_cost_at_end_time,omitempty"`
}

type ListCostMetricsResponse struct {
	TotalCount int          `json:"total_count" example:"10"`
	TotalCost  float64      `json:"total_cost" example:"1000"`
	Metrics    []CostMetric `json:"metrics"`
}

type ListCostCompositionResponse struct {
	TotalCount     int                `json:"total_count" example:"10"`
	TotalCostValue float64            `json:"total_cost_value" example:"1000"`
	TopValues      map[string]float64 `json:"top_values"`
	Others         float64            `json:"others" example:"100"`
}
