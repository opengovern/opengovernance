package kaytu_client

import (
	"context"
	"encoding/json"
	authApi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/steampipe-plugin-kaytu/kaytu-sdk/config"
	"github.com/kaytu-io/kaytu-engine/pkg/steampipe-plugin-kaytu/kaytu-sdk/services"
	"github.com/kaytu-io/kaytu-engine/pkg/utils"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"runtime"
	"time"
)

func GetBenchmarkSummary(ctx context.Context, d *plugin.QueryData, _ *plugin.HydrateData) (any, error) {
	plugin.Logger(ctx).Trace("GetBenchmarkSummary")
	runtime.GC()
	cfg := config.GetConfig(d.Connection)
	complianceClient, err := services.NewComplianceClientCached(cfg, d.ConnectionCache, ctx)
	if err != nil {
		return nil, err
	}

	benchmarkId := d.EqualsQuals["benchmark_id"].GetStringValue()

	var timeAt *time.Time
	if d.Quals["time_at"] != nil {
		timeAt = utils.GetPointer(d.EqualsQuals["time_at"].GetTimestampValue().AsTime())
	}
	var connectionIds []string
	if d.Quals["connection_ids"] != nil {
		jsonConnections := d.EqualsQuals["connection_ids"].GetJsonbValue()
		err := json.Unmarshal([]byte(jsonConnections), &connectionIds)
		if err != nil {
			plugin.Logger(ctx).Error("GetBenchmarkSummary connection id json conversion", "error", err)
			return nil, err
		}
	}

	res, err := complianceClient.GetBenchmarkSummary(&httpclient.Context{UserRole: authApi.InternalRole}, benchmarkId, connectionIds, timeAt)
	if err != nil {
		plugin.Logger(ctx).Error("GetBenchmarkSummary compliance client call failed", "error", err)
		return nil, err
	}

	return res, nil
}
