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
			"platform_findings":               tablePlatformFindings(ctx),
			"platform_resources":              tablePlatformResources(ctx),
			"platform_lookup":                 tablePlatformLookup(ctx),
			"platform_cost":                   tablePlatformCost(ctx),
			"platform_integrations":           tablePlatformConnections(ctx),
			"platform_metrics":                tablePlatformMetrics(ctx),
			"platform_api_benchmark_summary":  tablePlatformApiBenchmarkSummary(ctx),
			"platform_api_benchmark_controls": tablePlatformApiBenchmarkControls(ctx),
		},
	}

	viewSync := newViewSync()
	go viewSync.start(ctx)

	return p
}
