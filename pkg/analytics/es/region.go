package es

import (
	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"strconv"
)

const (
	AnalyticsRegionSummaryIndex = "analytics_region_summary"
)

type RegionMetricTrendSummary struct {
	Region        string      `json:"region"`
	ConnectionID  uuid.UUID   `json:"connection_id"`
	Connector     source.Type `json:"connector"`
	EvaluatedAt   int64       `json:"evaluated_at"`
	MetricID      string      `json:"metric_id"`
	ResourceCount int         `json:"resource_count"`
}

func (r RegionMetricTrendSummary) KeysAndIndex() ([]string, string) {
	keys := []string{
		r.Region,
		r.ConnectionID.String(),
		r.MetricID,
		strconv.FormatInt(r.EvaluatedAt, 10),
	}
	return keys, AnalyticsRegionSummaryIndex
}
