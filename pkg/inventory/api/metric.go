package api

import (
	analyticsDB "github.com/kaytu-io/kaytu-engine/pkg/analytics/db"
	"github.com/kaytu-io/kaytu-util/pkg/model"
	"github.com/kaytu-io/kaytu-util/pkg/source"
)

type Metric struct {
	Connectors []source.Type       `json:"connectors" example:"[Azure]"` // Cloud Provider
	Name       string              `json:"name" example:"VMs"`           // Resource Type
	Tags       map[string][]string `json:"tags,omitempty"`               // Tags

	Count    *int `json:"count" example:"100"`    // Number of Resources of this Resource Type - Metric
	OldCount *int `json:"old_count" example:"90"` // Number of Resources of this Resource Type in the past - Metric
}

func MetricToAPI(metric analyticsDB.AnalyticMetric) Metric {
	return Metric{
		Connectors: source.ParseTypes(metric.Connectors),
		Name:       metric.Name,
		Tags:       model.TrimPrivateTags(GetTagsMap(metric)),
	}
}

func GetTagsMap(r analyticsDB.AnalyticMetric) map[string][]string {
	tagLikeArr := make([]model.TagLike, 0, len(r.Tags))
	for _, tag := range r.Tags {
		tagLikeArr = append(tagLikeArr, tag)
	}
	return model.GetTagsMap(tagLikeArr)
}

type ListMetricsResponse struct {
	TotalCount    int      `json:"total_count"`
	TotalOldCount int      `json:"total_old_count"`
	TotalMetrics  int      `json:"total_metrics"`
	Metrics       []Metric `json:"metrics"`
}
