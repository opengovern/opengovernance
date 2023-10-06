package resource

import (
	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"strconv"
)

const (
	AnalyticsConnectionSummaryIndex                    = "analytics_connection_summary"
	ResourceCollectionsAnalyticsConnectionSummaryIndex = "rc_analytics_connection_summary"
)

type ConnectionMetricTrendSummary struct {
	ConnectionID    uuid.UUID   `json:"connection_id"`
	ConnectionName  string      `json:"connection_name"`
	Connector       source.Type `json:"connector"`
	EvaluatedAt     int64       `json:"evaluated_at"`
	Date            string      `json:"date"`
	Month           string      `json:"month"`
	Year            string      `json:"year"`
	MetricID        string      `json:"metric_id"`
	MetricName      string      `json:"metric_name"`
	ResourceCount   int         `json:"resource_count"`
	IsJobSuccessful bool        `json:"is_job_successful"`

	ResourceCollection *string `json:"resource_collection"`
}

func (r ConnectionMetricTrendSummary) KeysAndIndex() ([]string, string) {
	keys := []string{
		strconv.FormatInt(r.EvaluatedAt, 10),
		r.ConnectionID.String(),
		r.MetricID,
	}
	idx := AnalyticsConnectionSummaryIndex
	if r.ResourceCollection != nil {
		keys = append(keys, *r.ResourceCollection)
		idx = ResourceCollectionsAnalyticsConnectionSummaryIndex
	}
	return keys, idx
}
