package es

import (
	"github.com/kaytu-io/kaytu-engine/pkg/summarizer/es"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"strconv"
)

type ConnectorMetricTrendSummary struct {
	Connector     source.Type           `json:"connector"`
	EvaluatedAt   int64                 `json:"evaluated_at"`
	MetricID      string                `json:"metric_id"`
	ResourceCount int                   `json:"resource_count"`
	ReportType    es.ProviderReportType `json:"report_type"`
}

func (r ConnectorMetricTrendSummary) KeysAndIndex() ([]string, string) {
	keys := []string{
		r.Connector.String(),
		r.MetricID,
		strconv.FormatInt(r.EvaluatedAt, 10),
		string(es.MetricTrendConnectorSummary),
	}
	return keys, es.ProviderSummaryIndex
}
