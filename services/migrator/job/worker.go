package job

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgtype"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"github.com/kaytu-io/open-governance/services/migrator/config"
	"github.com/kaytu-io/open-governance/services/migrator/db"
	"github.com/kaytu-io/open-governance/services/migrator/db/model"
	"go.uber.org/zap"
	"time"
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

	m, err := w.db.GetMigration("main")
	if err != nil {
		return nil, err
	}

	if m == nil {
		jp := pgtype.JSONB{}
		err = jp.Set([]byte(""))
		if err != nil {
			return nil, err
		}
		m = &model.Migration{
			ID:             "main",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
			AdditionalInfo: "",
			Status:         "Fetching Data",
			JobsStatus:     jp,
		}
		err = w.db.CreateMigration(m)
		if err != nil {
			return nil, err
		}
	} else {
		jp := pgtype.JSONB{}
		err = jp.Set([]byte(""))
		if err != nil {
			return nil, err
		}
		err = w.db.UpdateMigrationJob("main", "Fetching Data", jp)
		if err != nil {
			return nil, err
		}
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

	m, err := w.db.GetMigration("main")
	if err != nil {
		return err
	}

	for name, mig := range migrations {
		w.logger.Info("running migration", zap.String("migrationName", name))
		//updateNeeded, err := w.CheckIfUpdateIsNeeded(name, mig)
		//if err != nil {
		//	w.logger.Error("failed to CheckIfUpdateIsNeeded", zap.Error(err), zap.String("migrationName", name))
		//	continue
		//}
		//
		//if !updateNeeded {
		//	w.logger.Info("migration is up to date", zap.String("migrationName", name))
		//	continue
		//}
		m.Status = fmt.Sprintf("Running migration %s", name)

		err = w.db.UpdateMigrationJob(m.ID, m.Status, m.JobsStatus)
		if err != nil {
			return err
		}

		updateFailed := false
		err := mig.Run(ctx, w.conf, w.logger)
		if err != nil {
			w.logger.Error("failed to run migration", zap.Error(err), zap.String("migrationName", name))
			updateFailed = true
		}

		jobsStatus := make(map[string]model.JobsStatus)

		if len(m.JobsStatus.Bytes) > 0 {
			err := json.Unmarshal(m.JobsStatus.Bytes, &jobsStatus)
			if err != nil {
				return err
			}
		}
		if updateFailed {
			jobsStatus[name] = model.JobStatusFailed
		} else {
			jobsStatus[name] = model.JobStatusCompleted
		}

		jobsStatusJson, err := json.Marshal(jobsStatus)
		if err != nil {
			return err
		}

		jp := pgtype.JSONB{}
		err = jp.Set(jobsStatusJson)
		if err != nil {
			return err
		}
		m.JobsStatus = jp

		err = w.db.UpdateMigrationJob(m.ID, m.Status, m.JobsStatus)
		if err != nil {
			return err
		}

		//if !updateFailed {
		//	err = w.UpdateMigration(name, mig)
		//	if err != nil {
		//		w.logger.Error("failed to update migration", zap.Error(err), zap.String("migrationName", name))
		//	}
		//}
	}

	err = w.db.UpdateMigrationJob(m.ID, "COMPLETED", m.JobsStatus)
	if err != nil {
		return err
	}

	return nil
}
