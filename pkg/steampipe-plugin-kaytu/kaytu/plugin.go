package kaytu

import (
	"context"
	steampipesdk "github.com/opengovern/og-util/pkg/steampipe"
	"github.com/opengovern/opengovernance/pkg/metadata/models"
	"github.com/opengovern/opengovernance/pkg/steampipe-plugin-kaytu/kaytu-sdk/pg"
	"github.com/opengovern/opengovernance/pkg/utils"
	"go.uber.org/zap"
	"os"
	"strings"
	"time"

	"github.com/opengovern/opengovernance/pkg/steampipe-plugin-kaytu/kaytu-sdk/config"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin"
	"github.com/turbot/steampipe-plugin-sdk/v5/plugin/transform"
)

func updateViewsInDatabase(ctx context.Context, selfClient *steampipesdk.SelfClient, metadataClient pg.Client) {
	var queryViews []models.QueryView

	err := metadataClient.DB().Find(&queryViews).Error
	if err != nil {
		logger.Error("Error fetching query views from metadata", zap.Error(err))
		logger.Sync()
		return
	}

initLoop:
	for i := 0; i < 60; i++ {
		time.Sleep(10 * time.Second)

		for _, view := range queryViews {
			dropQuery := "DROP MATERIALIZED VIEW IF EXISTS " + view.ID + " CASCADE"
			_, err := selfClient.GetConnection().Exec(ctx, dropQuery)
			if err != nil {
				logger.Error("Error dropping materialized view", zap.Error(err), zap.String("view", view.ID))
				logger.Sync()
				continue
			}

			query := "CREATE MATERIALIZED VIEW IF NOT EXISTS " + view.ID + " AS " + view.Query
			_, err = selfClient.GetConnection().Exec(ctx, query)
			if strings.Contains(err.Error(), "SQLSTATE 42P01") {
				continue initLoop
			}
			if err != nil {
				logger.Error("Error creating materialized view", zap.Error(err), zap.String("view", view.ID))
				logger.Sync()
				continue
			}
		}
	}
}

func newZapLogger() (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	cfg.OutputPaths = []string{
		"/home/steampipe/.steampipe/logs/kaytu.log",
	}
	return cfg.Build()
}

var logger, _ = newZapLogger()

func initViews(ctx context.Context) {
	logger.Info("Initializing materialized views")
	logger.Info("Creating self client")
	logger.Sync()
	selfClient, err := steampipesdk.NewSelfClient(ctx)
	if err != nil {
		logger.Error("Error creating self client for init materialized views", zap.Error(err))
		logger.Sync()
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
	logger.Info("Creating metadata client")
	logger.Sync()
	metadataClient, err := pg.NewMetadataClient(metadataClientConfig, ctx)
	if err != nil {
		logger.Error("Error creating metadata client for init materialized views", zap.Error(err))
		logger.Sync()
		return
	}

	updateViewsInDatabase(ctx, selfClient, metadataClient)

	selfClient.GetConnection().Close()
	db, _ := metadataClient.DB().DB()
	db.Close()

	ticker := time.NewTicker(2 * time.Hour)
	go func() {
		for range ticker.C {
			selfClient, err := steampipesdk.NewSelfClient(ctx)
			if err != nil {
				logger.Error("Error creating self client for refreshing materialized views", zap.Error(err))
				logger.Sync()
				continue
			}
			metadataClient, err := pg.NewMetadataClient(metadataClientConfig, ctx)
			if err != nil {
				logger.Error("Error creating metadata client for init materialized views", zap.Error(err))
				logger.Sync()
				return
			}
			if err != nil {
				logger.Error("Error creating metadata client for refreshing materialized views", zap.Error(err))
				logger.Sync()
				continue
			}
			updateViewsInDatabase(ctx, selfClient, metadataClient)
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
				logger.Error("Error creating RefreshAllMaterializedViews function", zap.Error(err))
				logger.Sync()
				continue
			}
			_, err = selfClient.GetConnection().Exec(ctx, "SELECT RefreshAllMaterializedViews('public')")
			if err != nil {
				logger.Error("Error refreshing materialized views", zap.Error(err))
				logger.Sync()
				continue
			}

			selfClient.GetConnection().Close()
			db, _ := metadataClient.DB().DB()
			db.Close()
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

	go initViews(ctx)

	return p
}
