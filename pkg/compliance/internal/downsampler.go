package internal

import (
	"github.com/haoel/downsampling/core"
	"gitlab.com/keibiengine/keibi-engine/pkg/compliance/api"
)

func insightTrendDataPointsToPoints(trendDataPoints []api.InsightTrendDatapoint) []core.Point {
	points := make([]core.Point, len(trendDataPoints))
	for i, trendDataPoint := range trendDataPoints {
		points[i] = core.Point{
			X: float64(trendDataPoint.Timestamp),
			Y: float64(trendDataPoint.Value),
		}
	}
	return points
}

func pointsToInsightTrendDataPoints(points []core.Point) []api.InsightTrendDatapoint {
	trendDataPoints := make([]api.InsightTrendDatapoint, len(points))
	for i, point := range points {
		trendDataPoints[i] = api.InsightTrendDatapoint{
			Timestamp: int(point.X),
			Value:     int(point.Y),
		}
	}
	return trendDataPoints
}

func DownSampleInsightTrendDatapoints(trendDataPoints []api.InsightTrendDatapoint, maxDataPoints int) []api.InsightTrendDatapoint {
	if len(trendDataPoints) <= maxDataPoints {
		return trendDataPoints
	}
	downSampledResourceCounts := core.LTTB(insightTrendDataPointsToPoints(trendDataPoints), maxDataPoints)
	return pointsToInsightTrendDataPoints(downSampledResourceCounts)
}
