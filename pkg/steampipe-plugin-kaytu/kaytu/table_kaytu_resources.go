package kaytu

import (
	"context"
	kaytu_client "github.com/kaytu-io/kaytu-engine/pkg/steampipe-plugin-kaytu/kaytu-client"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
	"time"
)

func tableKaytuResources(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "kaytu_resources",
		Description: "Kaytu Resources",
		Cache: &plugin.TableCacheOptions{
			Enabled: false,
		},
		List: &plugin.ListConfig{
			Hydrate: kaytu_client.ListResources,
			KeyColumns: []*plugin.KeyColumn{
				{
					Name:    "resource_type",
					Require: "required",
				},
			},
		},
		Columns: []*plugin.Column{
			{Name: "resource_id", Transform: transform.FromField("ID"), Type: proto.ColumnType_STRING},
			{Name: "resource_arn", Transform: transform.FromField("ARN"), Type: proto.ColumnType_STRING},
			{Name: "connection_id", Transform: transform.FromField("SourceID"), Type: proto.ColumnType_STRING},
			{Name: "connector", Transform: transform.FromField("SourceType"), Type: proto.ColumnType_STRING},
			{Name: "resource_type", Type: proto.ColumnType_STRING},
			{Name: "name", Transform: transform.FromField("Metadata.Name"), Type: proto.ColumnType_STRING},
			{Name: "region", Transform: transform.From(getResourceRegion), Type: proto.ColumnType_STRING},
			{Name: "created_at", Transform: transform.From(fixTime), Type: proto.ColumnType_TIMESTAMP},
			{Name: "description", Type: proto.ColumnType_JSON},
		},
	}
}

func fixTime(ctx context.Context, d *transform.TransformData) (interface{}, error) {
	resource := d.HydrateItem.(kaytu_client.Resource)
	t := time.UnixMilli(resource.CreatedAt)
	return t.Format("2006-01-02T15:04:05"), nil
}

func getResourceRegion(ctx context.Context, d *transform.TransformData) (interface{}, error) {
	resource := d.HydrateItem.(kaytu_client.Resource)
	if len(resource.Metadata.Region) > 0 {
		return resource.Metadata.Region, nil
	}
	return resource.Metadata.Location, nil
}
