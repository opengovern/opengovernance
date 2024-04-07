package job

import (
	"context"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/services/migrator/config"
	"github.com/kaytu-io/kaytu-engine/services/migrator/db"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"go.uber.org/zap"
)

type GitConfig struct {
	AnalyticsGitURL         string
	ControlEnrichmentGitURL string
	githubToken             string
}

type Job struct {
	db         db.Database
	logger     *zap.Logger
	conf       config.MigratorConfig
	commitRefs string
}

func InitializeJob(
	conf config.MigratorConfig,
	logger *zap.Logger,
) (w *Job, err error) {
	w = &Job{
		conf:   conf,
		logger: logger,
	}

	cfg := postgres.Config{
		Host:    conf.PostgreSQL.Host,
		Port:    conf.PostgreSQL.Port,
		User:    conf.PostgreSQL.Username,
		Passwd:  conf.PostgreSQL.Password,
		DB:      conf.PostgreSQL.DB,
		SSLMode: conf.PostgreSQL.SSLMode,
	}
	orm, err := postgres.NewClient(&cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("new postgres client: %w", err)
	}
	w.db = db.Database{ORM: orm}
	fmt.Println("Connected to the postgres database: ", conf.PostgreSQL.DB)

	err = w.db.Initialize()
	if err != nil {
		return nil, fmt.Errorf("failure while initializing database: %w", err)
	}

	w.commitRefs, err = GitClone(conf, logger)
	if err != nil {
		return nil, fmt.Errorf("failure while running git clone: %w", err)
	}

	return w, nil
}

func (w *Job) Run(ctx context.Context) error {
	defer func() {
		if r := recover(); r != nil {
			w.logger.Error("paniced with error", zap.Error(fmt.Errorf("%v", r)))
		}
	}()
	w.logger.Info("Starting migrator job")

	for name, mig := range migrations {
		updateNeeded, err := w.CheckIfUpdateIsNeeded(name, mig)
		if err != nil {
			w.logger.Error("failed to CheckIfUpdateIsNeeded", zap.Error(err), zap.String("migrationName", name))
			continue
		}

		if !updateNeeded {
			w.logger.Info("migration is up to date", zap.String("migrationName", name))
			continue
		}

		updateFailed := false
		err = mig.Run(ctx, w.conf, w.logger)
		if err != nil {
			w.logger.Error("failed to run migration", zap.Error(err), zap.String("migrationName", name))
			updateFailed = true
		}

		if !updateFailed {
			err = w.UpdateMigration(name, mig)
			if err != nil {
				w.logger.Error("failed to update migration", zap.Error(err), zap.String("migrationName", name))
			}
		}
	}

	return nil
}
