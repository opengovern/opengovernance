package opengovernance

import (
	"context"
	kaytu_client "github.com/opengovern/opengovernance/pkg/steampipe-plugin-opengovernance/opengovernance-client"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
)

func tableKaytuCostEstimate(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "pennywise_cost_estimate",
		Description: "Pennywise Resource Cost Estimate",
		Cache: &plugin.TableCacheOptions{
			Enabled: false,
		},
		List: &plugin.ListConfig{
			Hydrate: kaytu_client.ListResourceCostEstimate,
			KeyColumns: []*plugin.KeyColumn{
				{
					Name:      "resource_id",
					Operators: []string{"="},
					Require:   "required",
				},
				{
					Name:      "resource_type",
					Operators: []string{"="},
					Require:   "required",
				},
			},
		},
		Columns: []*plugin.Column{
			{Name: "resource_id", Type: proto.ColumnType_STRING},
			{Name: "resource_type", Type: proto.ColumnType_STRING},
			{Name: "cost", Type: proto.ColumnType_DOUBLE},
		},
	}
}
