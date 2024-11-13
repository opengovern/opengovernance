package opengovernance

import (
	"context"
	metric "github.com/opengovern/opengovernance/pkg/analytics/db"
	og_client "github.com/opengovern/opengovernance/pkg/steampipe-plugin-opengovernance/opengovernance-client"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

func tablePlatformMetrics(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "platform_metrics",
		Description: "opengovernance Metrics",
		Cache: &plugin.TableCacheOptions{
			Enabled: false,
		},
		List: &plugin.ListConfig{
			Hydrate: og_client.ListMetrics,
		},
		Columns: []*plugin.Column{
			{Name: "id", Type: proto.ColumnType_STRING, Transform: transform.FromField("ID")},
			{Name: "connectors", Type: proto.ColumnType_JSON, Transform: transform.FromField("Connectors")},
			{Name: "type", Type: proto.ColumnType_STRING, Transform: transform.FromField("Type")},
			{Name: "name", Type: proto.ColumnType_STRING, Transform: transform.FromField("Name")},
			{Name: "query", Type: proto.ColumnType_STRING, Transform: transform.FromField("Query")},
			{Name: "tables", Type: proto.ColumnType_JSON, Transform: transform.FromField("Tables")},
			{Name: "finderQuery", Type: proto.ColumnType_STRING, Transform: transform.FromField("FinderQuery")},
			{Name: "category", Type: proto.ColumnType_STRING, Transform: transform.From(getCategory)},
			{Name: "tags", Type: proto.ColumnType_JSON, Transform: transform.FromField("Tags")},
		},
	}
}

func getCategory(ctx context.Context, data *transform.TransformData) (interface{}, error) {
	m := data.HydrateItem.(metric.AnalyticMetric)
	category := ""
	for _, t := range m.Tags {
		if t.Key == "category" {
			for _, c := range t.Value {
				category = c
			}
		}
	}
	return category, nil
}
