package internal

import (
	"github.com/haoel/downsampling/core"
	"gitlab.com/keibiengine/keibi-engine/pkg/inventory/api"
)

func costsToPoints(costs []api.CostTrendDataPoint) []core.Point {
	points := make([]core.Point, len(costs))
	for i, cost := range costs {
		points[i] = core.Point{
			X: float64(cost.Timestamp),
			Y: cost.Value.Cost,
		}
	}
	return points
}

func pointsToCosts(points []core.Point, unit string) []api.CostTrendDataPoint {
	costs := make([]api.CostTrendDataPoint, len(points))
	for i, point := range points {
		costs[i] = api.CostTrendDataPoint{
			Timestamp: int64(point.X),
			Value: api.CostWithUnit{
				Cost: point.Y,
				Unit: unit,
			},
		}
	}
	return costs
}

func DownSampleCosts(costs map[string][]api.CostTrendDataPoint, maxDataPoints int) map[string][]api.CostTrendDataPoint {
	for unit, costsArr := range costs {
		if len(costsArr) <= maxDataPoints {
			continue
		}
		downSampledCosts := core.LTTB(costsToPoints(costsArr), maxDataPoints)
		costsArr = pointsToCosts(downSampledCosts, unit)
		costs[unit] = costsArr
	}
	return costs
}

func resourceCountsToPoints(resourceCounts []api.TrendDataPoint) []core.Point {
	points := make([]core.Point, len(resourceCounts))
	for i, resourceCount := range resourceCounts {
		points[i] = core.Point{
			X: float64(resourceCount.Timestamp),
			Y: float64(resourceCount.Value),
		}
	}
	return points
}

func pointsToResourceCounts(points []core.Point) []api.TrendDataPoint {
	resourceCounts := make([]api.TrendDataPoint, len(points))
	for i, point := range points {
		resourceCounts[i] = api.TrendDataPoint{
			Timestamp: int64(point.X),
			Value:     int64(point.Y),
		}
	}
	return resourceCounts
}

func DownSampleResourceCounts(resourceCounts []api.TrendDataPoint, maxDataPoints int) []api.TrendDataPoint {
	if len(resourceCounts) <= maxDataPoints {
		return resourceCounts
	}
	downSampledResourceCounts := core.LTTB(resourceCountsToPoints(resourceCounts), maxDataPoints)
	return pointsToResourceCounts(downSampledResourceCounts)
}
