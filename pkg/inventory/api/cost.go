package api

import (
	"time"

	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type CostTrendDatapoint struct {
	Cost float64   `json:"count" minimum:"0"`
	Date time.Time `json:"date" format:"date-time"`
}

type ListServicesCostTrendDatapoint struct {
	ServiceName string               `json:"serviceName" example:"EC2-Service-Example"`
	CostTrend   []CostTrendDatapoint `json:"costTrend"`
}

type CostMetric struct {
	Connector            []source.Type `json:"connector" example:"Azure"`
	CostDimensionName    string        `json:"cost_dimension_name" example:"microsoft.compute/disks"`
	TotalCost            *float64      `json:"total_cost,omitempty" example:"621041.2436112489" minimum:"0"`
	DailyCostAtStartTime *float64      `json:"daily_cost_at_start_time,omitempty" example:"21232.10443638001" minimum:"0"`
	DailyCostAtEndTime   *float64      `json:"daily_cost_at_end_time,omitempty" example:"14118.815231085681" minimum:"0"`
}

type ListCostMetricsResponse struct {
	TotalCount int          `json:"total_count" example:"10" minimum:"0"`
	TotalCost  float64      `json:"total_cost" example:"1000" minimum:"0"`
	Metrics    []CostMetric `json:"metrics"`
}

type ListCostCompositionResponse struct {
	TotalCount     int                `json:"total_count" example:"10" minimum:"0"`
	TotalCostValue float64            `json:"total_cost_value" example:"1000" minimum:"0"`
	TopValues      map[string]float64 `json:"top_values"`
	Others         float64            `json:"others" example:"100" minimum:"0"`
}

type SpendTableRow struct {
	Dimension string             `json:"dimension" example:"Compute"`
	CostValue map[string]float64 `json:"costValue"`
}
