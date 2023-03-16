package migrator

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus/push"
	"gitlab.com/keibiengine/keibi-engine/pkg/internal/postgres"
	"gitlab.com/keibiengine/keibi-engine/pkg/migrator/compliance"
	"gitlab.com/keibiengine/keibi-engine/pkg/migrator/db"
	"go.uber.org/zap"
	"os"
)

type Job struct {
	db     db.Database
	logger *zap.Logger
	pusher *push.Pusher
}

func InitializeJob(
	conf JobConfig,
	logger *zap.Logger,
	prometheusPushAddress string,
) (w *Job, err error) {

	w = &Job{
		logger: logger,
	}
	defer func() {
		if err != nil && w != nil {
			w.Stop()
		}
	}()

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

	w.pusher = push.New(prometheusPushAddress, "migrator")

	return w, nil
}

func (w *Job) Run() error {
	defer func() {
		if r := recover(); r != nil {
			w.logger.Error("paniced with error", zap.Error(fmt.Errorf("%v", r)))
		}
	}()

	w.logger.Info("Starting migrator job")
	if err := compliance.Run(w.db, ""); err != nil {
		w.logger.Error(fmt.Sprintf("Failure while running compliance migration: %v", err))
	}

	return nil
}

func (w *Job) Stop() {
	os.RemoveAll("/tmp/loader-input-git")
}
