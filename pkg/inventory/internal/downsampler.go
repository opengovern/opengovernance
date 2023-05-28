package internal

import (
	"time"

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

func trendDataPointsToPoints(trendDataPoints []api.TrendDataPoint) []core.Point {
	points := make([]core.Point, len(trendDataPoints))
	for i, trendDataPoint := range trendDataPoints {
		points[i] = core.Point{
			X: float64(trendDataPoint.Timestamp),
			Y: float64(trendDataPoint.Value),
		}
	}
	return points
}

func pointsToTrendDataPoints(points []core.Point) []api.TrendDataPoint {
	trendDataPoints := make([]api.TrendDataPoint, len(points))
	for i, point := range points {
		trendDataPoints[i] = api.TrendDataPoint{
			Timestamp: int64(point.X),
			Value:     int64(point.Y),
		}
	}
	return trendDataPoints
}

func DownSampleTrendDataPoints(trendDataPoints []api.TrendDataPoint, maxDataPoints int) []api.TrendDataPoint {
	if len(trendDataPoints) <= maxDataPoints {
		return trendDataPoints
	}
	downSampledResourceCounts := core.LTTB(trendDataPointsToPoints(trendDataPoints), maxDataPoints)
	return pointsToTrendDataPoints(downSampledResourceCounts)
}

func resourceTypeTrendDataPointsToPoints(trendDataPoints []api.ResourceTypeTrendDatapoint) []core.Point {
	points := make([]core.Point, len(trendDataPoints))
	for i, trendDataPoint := range trendDataPoints {
		points[i] = core.Point{
			X: float64(trendDataPoint.Date.UnixMilli()),
			Y: float64(trendDataPoint.Count),
		}
	}
	return points
}

func pointsToResourceTypeTrendDataPoints(points []core.Point) []api.ResourceTypeTrendDatapoint {
	trendDataPoints := make([]api.ResourceTypeTrendDatapoint, len(points))
	for i, point := range points {
		trendDataPoints[i] = api.ResourceTypeTrendDatapoint{
			Date:  time.UnixMilli(int64(point.X)),
			Count: int(point.Y),
		}
	}
	return trendDataPoints
}

func DownSampleResourceTypeTrendDatapoints(trendDataPoints []api.ResourceTypeTrendDatapoint, maxDataPoints int) []api.ResourceTypeTrendDatapoint {
	if len(trendDataPoints) <= maxDataPoints {
		return trendDataPoints
	}
	downSampledResourceCounts := core.LTTB(resourceTypeTrendDataPointsToPoints(trendDataPoints), maxDataPoints)
	return pointsToResourceTypeTrendDataPoints(downSampledResourceCounts)
}
