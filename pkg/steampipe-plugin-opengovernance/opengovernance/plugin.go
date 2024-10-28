package opengovernance

import (
	"context"
	"github.com/opengovern/opengovernance/pkg/steampipe-plugin-opengovernance/opengovernance-sdk/config"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

func Plugin(ctx context.Context) *plugin.Plugin {
	p := &plugin.Plugin{
		Name:             "steampipe-plugin-opengovernance",
		DefaultTransform: transform.FromGo().NullIfZero(),
		ConnectionConfigSchema: &plugin.ConnectionConfigSchema{
			NewInstance: config.Instance,
			Schema:      config.Schema(),
		},
		TableMap: map[string]*plugin.Table{
			"kaytu_findings":               tablePlatformFindings(ctx),
			"kaytu_resources":              tablePlatformResources(ctx),
			"kaytu_lookup":                 tablePlatformLookup(ctx),
			"kaytu_cost":                   tablePlatformCost(ctx),
			"pennywise_cost_estimate":      tablePlatformCostEstimate(ctx),
			"kaytu_connections":            tablePlatformConnections(ctx),
			"kaytu_metrics":                tablePlatformMetrics(ctx),
			"kaytu_api_benchmark_summary":  tablePlatformApiBenchmarkSummary(ctx),
			"kaytu_api_benchmark_controls": tablePlatformApiBenchmarkControls(ctx),
		},
	}

	viewSync := newViewSync()
	go viewSync.start(ctx)

	return p
}
