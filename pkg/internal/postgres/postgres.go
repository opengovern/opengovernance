package postgres

import (
	"errors"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormprom "gorm.io/plugin/prometheus"
)

const (
	defaultMaxOpenConns = 25
	defaultMaxIdleConns = 10
	defaultMaxLifetime  = 5 * time.Minute
)

type Config struct {
	Host   string
	Port   string
	User   string
	Passwd string
	DB     string

	Connection struct {
		MaxOpen     int
		MaxIdle     int
		MaxLifetime time.Duration
	}
}

func validateConfig(cfg *Config) error {
	if cfg.Host == "" {
		return errors.New("postgres host is empty")
	}
	if cfg.Port == "" {
		return errors.New("postgres port is empty")
	}
	if cfg.User == "" {
		return errors.New("postgres user is empty")
	}
	if cfg.Passwd == "" {
		return errors.New("postgres password is empty")
	}
	if cfg.DB == "" {
		return errors.New("postgres db is empty")
	}

	if cfg.Connection.MaxOpen == 0 {
		cfg.Connection.MaxOpen = defaultMaxOpenConns
	}
	if cfg.Connection.MaxIdle == 0 {
		cfg.Connection.MaxIdle = defaultMaxIdleConns
	}
	if cfg.Connection.MaxLifetime == 0 {
		cfg.Connection.MaxLifetime = defaultMaxLifetime
	}
	return nil
}

// NewClient will collect postgres metrics
//
// gorm_dbstats_max_open_connections: Maximum number of open connections to the database.
// gorm_dbstats_open_connections: The number of established connections both in use and idle.
// gorm_dbstats_in_use: The number of connections currently in use.
// gorm_dbstats_idle: The number of idle connections.
// gorm_dbstats_wait_count: The total number of connections waited for.
// gorm_dbstats_wait_duration: The total time blocked waiting for a new connection.
// gorm_dbstats_max_idle_closed: The total number of connections closed due to SetMaxIdleConns.
// gorm_dbstats_max_lifetime_closed: The total number of connections closed due to SetConnMaxLifetime.
func NewClient(cfg *Config, logger *zap.Logger) (*gorm.DB, error) {
	if cfg == nil {
		return nil, errors.New("cfg is nil")
	}
	if logger == nil {
		return nil, errors.New("logger is nil")
	}

	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	dsn := fmt.Sprintf(`host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=GMT`,
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Passwd,
		cfg.DB,
	)

	//orm, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
	//	Logger: zapgorm2.New(logger),
	//})
	orm, err := gorm.Open(postgres.Open(dsn))
	if err != nil {
		return nil, fmt.Errorf("gorm open: %w", err)
	}

	metrics := gormprom.New(gormprom.Config{
		DBName: cfg.DB,
	})
	if err := metrics.Initialize(orm); err != nil {
		return nil, fmt.Errorf("init gorm prometheus: %w", err)
	}
	for _, collector := range metrics.Collectors {
		prometheus.Register(collector)
	}

	db, err := orm.DB()
	if err != nil {
		return nil, fmt.Errorf("raw db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	db.SetMaxOpenConns(cfg.Connection.MaxOpen)
	db.SetMaxIdleConns(cfg.Connection.MaxIdle)
	db.SetConnMaxLifetime(cfg.Connection.MaxLifetime)

	return orm, nil
}
