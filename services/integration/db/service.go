package db

import (
	"fmt"

	"github.com/opengovern/og-util/pkg/koanf"
	"github.com/opengovern/og-util/pkg/postgres"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Database struct {
	DB *gorm.DB
}

func New(config koanf.Postgres, logger *zap.Logger) (Database, error) {
	cfg := postgres.Config{
		Host:    config.Host,
		Port:    config.Port,
		User:    config.Username,
		Passwd:  config.Password,
		DB:      config.DB,
		SSLMode: config.SSLMode,
	}
	gorm, err := postgres.NewClient(&cfg, logger.Named("postgres"))
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

	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetConnMaxIdleTime(config.ConnMaxIdleTime)
	sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)

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
