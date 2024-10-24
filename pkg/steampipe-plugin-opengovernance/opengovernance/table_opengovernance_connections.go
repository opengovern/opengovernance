package opengovernance

import (
	"context"

	kaytu_client "github.com/opengovern/opengovernance/pkg/steampipe-plugin-opengovernance/opengovernance-client"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

func tablePlatformConnections(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "kaytu_connections",
		Description: "Kaytu Connections",
		Cache: &plugin.TableCacheOptions{
			Enabled: false,
		},
		Get: &plugin.GetConfig{
			KeyColumns: plugin.AnyColumn([]string{"kaytu_id", "id"}),
			Hydrate:    kaytu_client.GetConnection,
		},
		List: &plugin.ListConfig{
			Hydrate: kaytu_client.ListConnections,
		},
		Columns: []*plugin.Column{
			{Name: "kaytu_id", Type: proto.ColumnType_STRING, Description: "The ID of the connection in Kaytu"},
			{Name: "id", Type: proto.ColumnType_STRING, Description: "The ID of the connection in the original connector"},
			{Name: "name", Type: proto.ColumnType_STRING, Description: "The name of the connection"},
			{Name: "connector", Type: proto.ColumnType_STRING},
			{Name: "health_state", Type: proto.ColumnType_STRING},
			{Name: "lifecycle_state", Type: proto.ColumnType_STRING},
			{Name: "tags", Type: proto.ColumnType_JSON, Transform: transform.FromJSONTag()},
		},
	}
}
