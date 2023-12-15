package db

import (
	"fmt"

	"github.com/kaytu-io/kaytu-util/pkg/config"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Database struct {
	DB *gorm.DB
}

func New(config config.Postgres, logger *zap.Logger) (Database, error) {
	cfg := postgres.Config{
		Host:    config.Host,
		Port:    config.Port,
		User:    config.Username,
		Passwd:  config.Password,
		DB:      config.DB,
		SSLMode: config.SSLMode,
	}
	gorm, err := postgres.NewClient(&cfg, logger)
	if err != nil {
		return Database{}, fmt.Errorf("new postgres client: %w", err)
	}

	db := Database{
		DB: gorm,
	}

	sqlDB, err := db.DB.DB()
	if err != nil {
		return Database{}, err
	}

	sqlDB.SetMaxIdleConns(0)
	sqlDB.SetMaxOpenConns(0)
	sqlDB.SetConnMaxIdleTime(0)
	sqlDB.SetConnMaxLifetime(0)

	if err := db.Initialize(); err != nil {
		return Database{}, err
	}

	return db, nil
}

func (db Database) Ping() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}

	if err := sqlDB.Ping(); err != nil {
		return err
	}

	return nil
}

func (db Database) Initialize() error {
	if err := db.DB.AutoMigrate(); err != nil {
		return err
	}

	return nil
}
