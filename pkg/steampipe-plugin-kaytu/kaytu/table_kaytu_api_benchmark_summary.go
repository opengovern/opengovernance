package kaytu

import (
	"context"
	compliance "github.com/kaytu-io/kaytu-engine/pkg/compliance/api"
	kaytu_client "github.com/kaytu-io/kaytu-engine/pkg/steampipe-plugin-kaytu/kaytu-client"
	"github.com/turbot/steampipe-plugin-sdk/v5/grpc/proto"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/quals"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
	"time"
)

func tableKaytuApiBenchmarkSummary(_ context.Context) *plugin.Table {
	return &plugin.Table{
		Name:        "kaytu_api_benchmark_summary",
		Description: "Wrapper for Kaytu benchmark summary api",
		Cache: &plugin.TableCacheOptions{
			Enabled: false,
		},
		Get: &plugin.GetConfig{
			KeyColumns: []*plugin.KeyColumn{
				{
					Name:      "benchmark_id",
					Operators: []string{quals.QualOperatorEqual},
					Require:   plugin.Required,
				},
				{
					Name:      "time_at",
					Operators: []string{quals.QualOperatorEqual},
					Require:   plugin.Optional,
				},
				{
					Name:      "connection_ids",
					Operators: []string{quals.QualOperatorEqual},
					Require:   plugin.Optional,
				},
			},
			Hydrate: kaytu_client.GetBenchmarkSummary,
		},
		Columns: []*plugin.Column{
			{
				Name:        "benchmark_id",
				Type:        proto.ColumnType_STRING,
				Description: "The ID of the benchmark in Kaytu",
				Transform:   transform.FromQual("benchmark_id"),
			},
			{
				Name:        "connection_ids",
				Type:        proto.ColumnType_JSON,
				Description: "The connection IDs included in the benchmark summary",
				Transform:   transform.FromQual("connection_ids"),
				Default:     "[]",
			},
			{
				Name:        "time_at",
				Type:        proto.ColumnType_TIMESTAMP,
				Description: "The timestamp of the benchmark summary record",
				Transform:   transform.FromQual("time_at"),
				Default:     time.Now(),
			},
			{
				Name:        "result",
				Type:        proto.ColumnType_JSON,
				Description: "The result of the benchmark summary (TEMPORARY for testing purposes)",
				Transform:   transform.From(getKaytuApiBenchmarkSummaryResult),
			},
		},
	}
}

func getKaytuApiBenchmarkSummaryResult(_ context.Context, d *transform.TransformData) (any, error) {
	return d.Value.(*compliance.BenchmarkEvaluationSummary), nil
}
