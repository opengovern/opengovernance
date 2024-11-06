package opengovernance

import (
	"context"

	og_client "github.com/opengovern/opengovernance/pkg/steampipe-plugin-opengovernance/opengovernance-client"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

func tablePlatformConnections(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "og_integrations",
		Description: "OpenGovernance Integrations",
		Cache: &plugin.TableCacheOptions{
			Enabled: false,
		},
		Get: &plugin.GetConfig{
			KeyColumns: plugin.AnyColumn([]string{"og_id", "id"}),
			Hydrate:    og_client.GetIntegration,
		},
		List: &plugin.ListConfig{
			Hydrate: og_client.ListIntegrations,
		},
		Columns: []*plugin.Column{
			{Name: "integration_id", Type: proto.ColumnType_STRING, Description: "The ID of the integration in OpenGovernance"},
			{Name: "provider_id", Type: proto.ColumnType_STRING, Description: "The ID of the integration in the provider"},
			{Name: "name", Type: proto.ColumnType_STRING, Description: "The name of the connection"},
			{Name: "integration_type", Type: proto.ColumnType_STRING},
			{Name: "health_state", Type: proto.ColumnType_STRING},
			{Name: "lifecycle_state", Type: proto.ColumnType_STRING},
			{Name: "tags", Type: proto.ColumnType_JSON, Transform: transform.FromJSONTag()},
		},
	}
}
