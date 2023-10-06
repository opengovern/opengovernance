package resource

import (
	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"strconv"
)

const (
	AnalyticsRegionSummaryIndex                    = "analytics_region_summary"
	ResourceCollectionsAnalyticsRegionSummaryIndex = "rc_analytics_region_summary"
)

type RegionMetricTrendSummary struct {
	Region         string      `json:"region"`
	ConnectionID   uuid.UUID   `json:"connection_id"`
	ConnectionName string      `json:"connection_name"`
	Connector      source.Type `json:"connector"`
	EvaluatedAt    int64       `json:"evaluated_at"`
	Date           string      `json:"date"`
	Month          string      `json:"month"`
	Year           string      `json:"year"`
	MetricID       string      `json:"metric_id"`
	MetricName     string      `json:"metric_name"`
	ResourceCount  int         `json:"resource_count"`

	ResourceCollection *string `json:"resource_collection"`
}

func (r RegionMetricTrendSummary) KeysAndIndex() ([]string, string) {
	keys := []string{
		r.Region,
		r.ConnectionID.String(),
		r.MetricID,
		strconv.FormatInt(r.EvaluatedAt, 10),
	}
	idx := AnalyticsRegionSummaryIndex
	if r.ResourceCollection != nil {
		keys = append(keys, *r.ResourceCollection)
		idx = ResourceCollectionsAnalyticsRegionSummaryIndex
	}
	return keys, idx
}
