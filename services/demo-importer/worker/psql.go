package worker

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kaytu-io/open-governance/services/demo-importer/types"
	"os"
)

func ImportPsqlData(ctx context.Context, cnf types.DemoImporterConfig, dataPath string) error {
	dbHost := cnf.PostgreSQL.Host
	dbPort := cnf.PostgreSQL.Port
	dbUser := cnf.PostgreSQL.Username
	dbPass := cnf.PostgreSQL.Password
	dbNames := []string{"pennywise", "workspace", "auth", "migrator", "describe", "onboard", "inventory", "compliance", "metadata"}

	for _, dbName := range dbNames {
		filePath := fmt.Sprintf("%s/%s.sql", dataPath, dbName)

		connStr := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s", dbUser, dbPass, dbHost, dbPort, dbName)

		config, err := pgxpool.ParseConfig(connStr)
		if err != nil {
			return err
		}

		dbPool, err := pgxpool.NewWithConfig(context.Background(), config)
		if err != nil {
			return err
		}

		config.ConnConfig.Database = dbName

		dbPool, err = pgxpool.NewWithConfig(context.Background(), config)
		if err != nil {
			return err
		}

		err = runSQLFile(ctx, dbPool, filePath)
		if err != nil {
			return err
		} else {
			fmt.Println("Successfully imported data for ", dbName)
		}

		dbPool.Close()
	}
	return nil
}

func runSQLFile(ctx context.Context, db *pgxpool.Pool, filePath string) error {
	sqlBytes, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read SQL file: %v", err)
	}

	sql := string(sqlBytes)

	_, err = db.Exec(ctx, sql)
	if err != nil {
		return fmt.Errorf("failed to execute SQL file: %v", err)
	}

	return nil
}
