package rego

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"
	"github.com/open-policy-agent/opa/types"
	steampipesdk "github.com/opengovern/og-util/pkg/steampipe"
	"go.uber.org/zap"
	"os"
	"time"
)

type RegoEngine struct {
	logger     *zap.Logger
	db         *pgxpool.Pool
	httpServer *echo.Echo

	regoFunctions []func(*rego.Rego)
}

var excludedTableSchema = []string{"information_schema", "pg_catalog", "steampipe_internal", "steampipe_command", "public"}

func NewRegoEngine(ctx context.Context, logger *zap.Logger) {
	engine := RegoEngine{
		logger: logger,
	}
	option := steampipesdk.GetDefaultSteampipeOption()
	selfClientConfig, err := pgxpool.ParseConfig(fmt.Sprintf(`host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=GMT`, option.Host, option.Port, option.User, option.Pass, option.Db))
	if err != nil {
		logger.Error("Unable to parse config", zap.Error(err))
		logger.Sync()
		return
	}
	logger.Info("Connecting to database", zap.String("host", option.Host), zap.String("port", option.Port), zap.String("user", option.User), zap.String("db", option.Db))
	logger.Sync()

	tries := 5
	for i := 0; i < tries; i++ {
		db, err := pgxpool.NewWithConfig(ctx, selfClientConfig)
		if err != nil {
			logger.Error("Unable to connect to database", zap.Error(err), zap.Int("try", i+1))
			logger.Sync()
			if i == tries-1 {
				logger.Error("Exhausted all tries to connect to database")
				logger.Sync()
				return
			}
			time.Sleep(10 * time.Second)
			continue
		}
		engine.db = db
		break
	}
	logger.Info("Connected to database")
	logger.Sync()

	tries = 5
	for i := 0; i < tries; i++ {
		functions, err := engine.getRegoFunctionForTables(ctx)
		if err != nil {
			logger.Error("Error getting rego functions", zap.Error(err), zap.Int("try", i+1))
			logger.Sync()
			if i == tries-1 {
				logger.Error("Exhausted all tries to get rego functions")
				logger.Sync()
				return
			}
			time.Sleep(10 * time.Second)
			continue
		}
		engine.regoFunctions = functions
		break
	}
	logger.Info("Got rego functions")
	logger.Sync()

	port := os.Getenv("REGO_PORT")
	if port == "" {
		port = "8001"
	}

	// use echo
	engine.httpServer = echo.New()
	engine.httpServer.Use(middleware.Recover())
	engine.httpServer.POST("/evaluate", engine.evaluateEndpoint)

	logger.Info("Starting rego server", zap.String("port", port))
	logger.Sync()
	err = engine.httpServer.Start(fmt.Sprintf("0.0.0.0:%s", port))
	if err != nil {
		logger.Error("Error starting rego server", zap.Error(err))
		logger.Sync()
	}
}

func (r *RegoEngine) getRegoFunctionForTables(ctx context.Context) ([]func(*rego.Rego), error) {

	rows, err := r.db.Query(ctx, "SELECT table_name FROM information_schema.tables WHERE table_schema != any ($1)", excludedTableSchema)
	if err != nil {
		r.logger.Error("Unable to query database", zap.Error(err))
		r.logger.Sync()
		return nil, err
	}
	defer rows.Close()

	results := make([]func(*rego.Rego), 0)
	for rows.Next() {
		var tableName string
		err := rows.Scan(&tableName)
		if err != nil {
			r.logger.Error("Unable to scan table name", zap.Error(err))
			r.logger.Sync()
			return nil, err
		}

		f := rego.FunctionDyn(&rego.Function{
			Name:             fmt.Sprintf("opencomply.%s", tableName),
			Description:      "",
			Decl:             types.NewFunction(nil, types.Any{}),
			Memoize:          true,
			Nondeterministic: true,
		}, func(bctx rego.BuiltinContext, terms []*ast.Term) (*ast.Term, error) {
			rows, err := r.db.Query(bctx.Context, fmt.Sprintf("SELECT * FROM %s", tableName))
			if err != nil {
				r.logger.Error("Unable to query database", zap.Error(err), zap.String("table", tableName))
				r.logger.Sync()
				return nil, err
			}
			defer rows.Close()

			results, err := pgx.CollectRows(rows, pgx.RowToMap)
			if err != nil {
				r.logger.Error("Unable to scan row", zap.Error(err), zap.String("table", tableName))
				r.logger.Sync()
				return nil, err
			}

			value, err := ast.InterfaceToValue(results)
			if err != nil {
				r.logger.Error("Unable to convert to value", zap.Error(err), zap.String("table", tableName))
				r.logger.Sync()
				return nil, err
			}

			return ast.NewTerm(value), nil
		})

		results = append(results, f)
	}

	return results, nil
}

func (r *RegoEngine) evaluate(ctx context.Context, policies []string, query string) (rego.ResultSet, error) {
	params := append(r.regoFunctions, rego.Query(query))
	for i, policy := range policies {
		params = append(params, rego.Module(fmt.Sprintf("policy_%d.rego", i+1), policy))
	}

	regoEngine := rego.New(params...)
	results, err := regoEngine.Eval(ctx)
	if err != nil {
		r.logger.Error("Error evaluating policy", zap.Error(err))
		r.logger.Sync()
		return nil, err
	}

	return results, nil
}
