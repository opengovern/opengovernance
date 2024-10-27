package opengovernance

import (
	"context"
	og_client "github.com/opengovern/opengovernance/pkg/steampipe-plugin-opengovernance/opengovernance-client"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

func tablePlatformLookup(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "og_lookup",
		Description: "OpenGovernance Resource Lookup",
		Cache: &plugin.TableCacheOptions{
			Enabled: false,
		},
		List: &plugin.ListConfig{
			Hydrate: og_client.ListLookupResources,
		},
		Columns: []*plugin.Column{
			{Name: "resource_id", Type: proto.ColumnType_STRING},
			{Name: "name", Type: proto.ColumnType_STRING},
			{Name: "connector", Transform: transform.FromField("SourceType"), Type: proto.ColumnType_STRING},
			{Name: "resource_type", Type: proto.ColumnType_STRING},
			{Name: "resource_group", Type: proto.ColumnType_STRING},
			{Name: "region", Transform: transform.FromField("Location"), Type: proto.ColumnType_STRING},
			{Name: "connection_id", Transform: transform.FromField("SourceID"), Type: proto.ColumnType_STRING},
			{Name: "created_at", Type: proto.ColumnType_INT},
			{Name: "tags", Type: proto.ColumnType_JSON},
		},
	}
}
