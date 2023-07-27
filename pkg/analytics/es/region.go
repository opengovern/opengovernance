package es

import (
	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-engine/pkg/summarizer/es"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"strconv"
)

type RegionMetricTrendSummary struct {
	Region        string                `json:"region"`
	ConnectionID  uuid.UUID             `json:"connection_id"`
	Connector     source.Type           `json:"connector"`
	EvaluatedAt   int64                 `json:"evaluated_at"`
	MetricID      string                `json:"metric_id"`
	ResourceCount int                   `json:"resource_count"`
	ReportType    es.ProviderReportType `json:"report_type"`
}

func (r RegionMetricTrendSummary) KeysAndIndex() ([]string, string) {
	keys := []string{
		r.Region,
		r.ConnectionID.String(),
		r.MetricID,
		strconv.FormatInt(r.EvaluatedAt, 10),
		string(es.MetricTrendRegionSummary),
	}
	return keys, es.ProviderSummaryIndex
}
