package es

import (
	"github.com/google/uuid"
	"github.com/kaytu-io/kaytu-engine/pkg/summarizer/es"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"strconv"
)

type ConnectionMetricTrendSummary struct {
	ConnectionID  uuid.UUID               `json:"connection_id"`
	Connector     source.Type             `json:"connector"`
	EvaluatedAt   int64                   `json:"evaluated_at"`
	MetricName    string                  `json:"metric_name"`
	ResourceCount int                     `json:"resource_count"`
	ReportType    es.ConnectionReportType `json:"report_type"`
}

func (r ConnectionMetricTrendSummary) KeysAndIndex() ([]string, string) {
	keys := []string{
		strconv.FormatInt(r.EvaluatedAt, 10),
		r.ConnectionID.String(),
		r.MetricName,
		string(es.MetricTrendConnectionSummary),
	}
	return keys, es.ConnectionSummaryIndex
}
