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
			"og_findings":               tablePlatformFindings(ctx),
			"og_resources":              tablePlatformResources(ctx),
			"og_lookup":                 tablePlatformLookup(ctx),
			"og_cost":                   tablePlatformCost(ctx),
			"pennywise_cost_estimate":   tablePlatformCostEstimate(ctx),
			"og_integrations":           tablePlatformConnections(ctx),
			"og_metrics":                tablePlatformMetrics(ctx),
			"og_api_benchmark_summary":  tablePlatformApiBenchmarkSummary(ctx),
			"og_api_benchmark_controls": tablePlatformApiBenchmarkControls(ctx),
		},
	}

	viewSync := newViewSync()
	go viewSync.start(ctx)

	return p
}
