package kaytu

import (
	"context"
	"github.com/hashicorp/go-hclog"
	steampipesdk "github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"github.com/kaytu-io/open-governance/pkg/metadata/models"
	"github.com/kaytu-io/open-governance/pkg/steampipe-plugin-kaytu/kaytu-sdk/pg"
	"github.com/kaytu-io/open-governance/pkg/utils"
	"os"
	"time"

	"github.com/kaytu-io/open-governance/pkg/steampipe-plugin-kaytu/kaytu-sdk/config"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

func updateViewsInDatabase(ctx context.Context, p *plugin.Plugin, selfClient *steampipesdk.SelfClient, metadataClient pg.Client) {
	var queryViews []models.QueryView

	err := metadataClient.DB().Find(&queryViews).Error
	if err != nil {
		p.Logger.Error("Error fetching query views from metadata", err)
		return
	}

	for _, view := range queryViews {
		dropQuery := "DROP MATERIALIZED VIEW IF EXISTS " + view.ID + " CASCADE"
		_, err := selfClient.GetConnection().Exec(ctx, dropQuery)
		if err != nil {
			p.Logger.Error("Error dropping materialized view", err, "view", view.ID)
			continue
		}

		query := "CREATE MATERIALIZED VIEW IF NOT EXISTS" + view.ID + " AS " + view.Query
		_, err = selfClient.GetConnection().Exec(ctx, query)
		if err != nil {
			p.Logger.Error("Error creating materialized view", err, "view", view.ID)
			continue
		}
	}
}

func initViews(ctx context.Context, p *plugin.Plugin) {
	selfClient, err := steampipesdk.NewSelfClient(ctx)
	if err != nil {
		p.Logger.Error("Error creating self client for init materialized views", err)
		return
	}
	metadataClientConfig := config.ClientConfig{
		PgHost:     utils.GetPointer(os.Getenv("METADATA_DB_HOST")),
		PgPort:     utils.GetPointer(os.Getenv("METADATA_DB_PORT")),
		PgPassword: utils.GetPointer(os.Getenv("PG_PASSWORD")),
		PgSslMode:  utils.GetPointer(os.Getenv("METADATA_DB_SSL_MODE")),
		PgUser:     utils.GetPointer("steampipe_user"),
		PgDatabase: utils.GetPointer("metadata"),
	}
	metadataClient, err := pg.NewMetadataClient(metadataClientConfig, ctx)
	if err != nil {
		p.Logger.Error("Error creating metadata client for init materialized views", err)
		return
	}

	updateViewsInDatabase(ctx, p, selfClient, metadataClient)

	ticker := time.NewTicker(2 * time.Hour)
	go func() {
		for range ticker.C {
			selfClient, err := steampipesdk.NewSelfClient(ctx)
			if err != nil {
				p.Logger.Error("Error creating self client for refreshing materialized views", err)
				continue
			}
			metadataClient, err := pg.NewMetadataClient(metadataClientConfig, ctx)
			if err != nil {
				p.Logger.Error("Error creating metadata client for init materialized views", err)
				return
			}
			if err != nil {
				p.Logger.Error("Error creating metadata client for refreshing materialized views", err)
				continue
			}
			updateViewsInDatabase(ctx, p, selfClient, metadataClient)
			query := `CREATE OR REPLACE FUNCTION RefreshAllMaterializedViews(schema_arg TEXT DEFAULT 'public')
RETURNS INT AS $$
DECLARE
    r RECORD;

BEGIN
    RAISE NOTICE 'Refreshing materialized view in schema %', schema_arg;
    if pg_is_in_recovery()  then 
    return 1;
    else
    FOR r IN SELECT matviewname FROM pg_matviews WHERE schemaname = schema_arg 
    LOOP
        RAISE NOTICE 'Refreshing %.%', schema_arg, r.matviewname;
        EXECUTE 'REFRESH MATERIALIZED VIEW ' || schema_arg || '.' || r.matviewname; 
    END LOOP;
    end if;
    RETURN 1;
END 
$$ LANGUAGE plpgsql;`

			_, err = selfClient.GetConnection().Exec(ctx, query)
			if err != nil {
				p.Logger.Error("Error creating RefreshAllMaterializedViews function", err)
				continue
			}
			_, err = selfClient.GetConnection().Exec(ctx, "SELECT RefreshAllMaterializedViews('public')")
			if err != nil {
				p.Logger.Error("Error refreshing materialized views", err)
				continue
			}
		}
	}()
}

func Plugin(ctx context.Context) *plugin.Plugin {
	p := &plugin.Plugin{
		Name:             "steampipe-plugin-kaytu",
		DefaultTransform: transform.FromGo().NullIfZero(),
		ConnectionConfigSchema: &plugin.ConnectionConfigSchema{
			NewInstance: config.Instance,
			Schema:      config.Schema(),
		},
		TableMap: map[string]*plugin.Table{
			"kaytu_findings":               tableKaytuFindings(ctx),
			"kaytu_resources":              tableKaytuResources(ctx),
			"kaytu_lookup":                 tableKaytuLookup(ctx),
			"kaytu_cost":                   tableKaytuCost(ctx),
			"pennywise_cost_estimate":      tableKaytuCostEstimate(ctx),
			"kaytu_connections":            tableKaytuConnections(ctx),
			"kaytu_metrics":                tableKaytuMetrics(ctx),
			"kaytu_api_benchmark_summary":  tableKaytuApiBenchmarkSummary(ctx),
			"kaytu_api_benchmark_controls": tableKaytuApiBenchmarkControls(ctx),
		},
	}
	if p.Logger == nil {
		outputFilePath := "~/.steampipe/log/kaytu.log"
		outputFile, err := os.OpenFile(outputFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil
		}
		p.Logger = hclog.New(&hclog.LoggerOptions{
			Output: outputFile,
		})
	}

	initViews(ctx, p)

	return p
}
