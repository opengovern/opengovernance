package connector

import (
	"fmt"
	"github.com/kaytu-io/kaytu-util/pkg/koanf"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"time"
)

type Database struct {
	db *gorm.DB
}

func New(config koanf.Postgres, logLevel logger.LogLevel) (*Database, error) {
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second * 10, // Slow SQL threshold
			LogLevel:                  logLevel,         // Log level
			IgnoreRecordNotFoundError: true,             // Ignore ErrRecordNotFound error for logger
			ParameterizedQueries:      false,            // Don't include params in the SQL log
			Colorful:                  true,             // Disable color
		},
	)
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s", config.Host, config.Username, config.Password, config.DB, config.Port, config.SSLMode)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: newLogger,
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
