package kaytu

import (
	"context"

	"github.com/kaytu-io/kaytu-engine/pkg/steampipe-plugin-kaytu/kaytu-sdk/config"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

func Plugin(ctx context.Context) *plugin.Plugin {
	p := &plugin.Plugin{
		Name:             "steampipe-plugin-kaytu",
		DefaultTransform: transform.FromGo().NullIfZero(),
		ConnectionConfigSchema: &plugin.ConnectionConfigSchema{
			NewInstance: config.Instance,
			Schema:      config.Schema(),
		},
		TableMap: map[string]*plugin.Table{
			"kaytu_findings":    tableKaytuFindings(ctx),
			"kaytu_resources":   tableKaytuResources(ctx),
			"kaytu_lookup":      tableKaytuLookup(ctx),
			"kaytu_cost":        tableKaytuCost(ctx),
			"kaytu_connections": tableKaytuConnections(ctx),
			"kaytu_metrics":     tableKaytuMetrics(ctx),
		},
	}
	return p
}
