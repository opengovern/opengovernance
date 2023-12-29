package db

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/services/subscription/db/model"
	"github.com/kaytu-io/kaytu-util/pkg/koanf"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Database struct {
	Orm *gorm.DB
}

func NewDatabase(config koanf.Postgres, logger *zap.Logger) (Database, error) {
	cfg := postgres.Config{
		Host:    config.Host,
		Port:    config.Port,
		User:    config.Username,
		Passwd:  config.Password,
		DB:      config.DB,
		SSLMode: config.SSLMode,
	}
	orm, err := postgres.NewClient(&cfg, logger)
	if err != nil {
		return Database{}, fmt.Errorf("new postgres client: %w", err)
	}

	db := Database{
		Orm: orm,
	}

	err = db.Initialize()
	if err != nil {
		return Database{}, err
	}

	return db, nil
}

func (db Database) Initialize() error {
	err := db.Orm.AutoMigrate(
		&model.Subscription{},
		&model.Meter{},
	)
	if err != nil {
		return err
	}

	return nil
}
