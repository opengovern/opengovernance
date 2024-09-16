package kaytu_client

import (
	"context"
	metric "github.com/kaytu-io/open-governance/pkg/analytics/db"
	"github.com/kaytu-io/open-governance/pkg/steampipe-plugin-kaytu/kaytu-sdk/config"
	"github.com/kaytu-io/open-governance/pkg/steampipe-plugin-kaytu/kaytu-sdk/pg"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"gorm.io/gorm/clause"
	"runtime"
)

func ListMetrics(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (any, error) {
	plugin.Logger(ctx).Trace("ListMetrics")
	runtime.GC()
	cfg := config.GetConfig(d.Connection)
	ke, err := pg.NewInventoryClientCached(cfg, d.ConnectionCache, ctx)
	if err != nil {
		plugin.Logger(ctx).Error("ListMetrics NewInventoryClientCached", "error", err)
		return nil, err
	}

	pdb := ke.DB()
	var s []metric.AnalyticMetric
	tx := pdb.Model(metric.AnalyticMetric{}).Preload(clause.Associations).Find(&s)

	if tx.Error != nil {
		plugin.Logger(ctx).Error("ListMetrics Find", "error", tx.Error)
		return nil, tx.Error
	}
	for _, v := range s {
		d.StreamListItem(ctx, v)
	}

	return s, nil
}
