package api

import (
	"github.com/kaytu-io/kaytu-engine/pkg/analytics/db"
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

type AnalyticsMetric struct {
	ID          string              `json:"id"`
	Connectors  []source.Type       `json:"connectors"`
	Type        db.MetricType       `json:"type"`
	Name        string              `json:"name"`
	Query       string              `json:"query"`
	Tables      []string            `json:"tables"`
	FinderQuery string              `json:"finderQuery"`
	Tags        map[string][]string `json:"tags"`
}

type ListCostCompositionResponse struct {
	TotalCount     int                `json:"total_count" example:"10" minimum:"0"`
	TotalCostValue float64            `json:"total_cost_value" example:"1000" minimum:"0"`
	TopValues      map[string]float64 `json:"top_values"`
	Others         float64            `json:"others" example:"100" minimum:"0"`
}

type SpendTableGranularity string

const (
	SpendTableGranularityDaily   SpendTableGranularity = "daily"
	SpendTableGranularityMonthly SpendTableGranularity = "monthly"
	SpendTableGranularityYearly  SpendTableGranularity = "yearly"
)

type SpendDimension string

const (
	SpendDimensionMetric     SpendDimension = "metric"
	SpendDimensionConnection SpendDimension = "connection"
)

type SpendTableRow struct {
	DimensionID   string             `json:"dimensionId" example:"compute"`
	AccountID     string             `json:"accountID" example:"1239042"`
	Connector     source.Type        `json:"connector" example:"AWS"`
	Category      string             `json:"category" example:"Compute"`
	DimensionName string             `json:"dimensionName" example:"Compute"`
	CostValue     map[string]float64 `json:"costValue"`
}

type AssetTableRow struct {
	DimensionID   string             `json:"dimensionId" example:"compute"`
	DimensionName string             `json:"dimensionName" example:"Compute"`
	ResourceCount map[string]float64 `json:"resourceCount"`
}
