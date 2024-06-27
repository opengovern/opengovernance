package connector

import (
	"fmt"
	"github.com/kaytu-io/kaytu-util/pkg/koanf"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"moul.io/zapgorm2"
)

type Database struct {
	db *gorm.DB
}

func New(config koanf.Postgres, logger *zap.Logger, logLevel logger.LogLevel) (*Database, error) {
	gormLogger := zapgorm2.New(logger).LogMode(logLevel)
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s", config.Host, config.Username, config.Password, config.DB, config.Port, config.SSLMode)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(10)

	return &Database{
		db: db,
	}, nil
}

func (s *Database) Conn() *gorm.DB {
	return s.db
}
