package neo4j

import (
	"errors"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.uber.org/zap"
)

type Config struct {
	Host   string
	Port   string
	User   string
	Passwd string
}

func validateConfig(cfg *Config) error {
	if cfg.Host == "" {
		return errors.New("neo4j host is empty")
	}
	if cfg.Port == "" {
		return errors.New("neo4j port is empty")
	}
	if cfg.User == "" {
		return errors.New("neo4j user is empty")
	}
	if cfg.Passwd == "" {
		return errors.New("neo4j password is empty")
	}

	return nil
}

func NewDriver(cfg *Config, logger *zap.Logger) (neo4j.DriverWithContext, error) {
	if cfg == nil {
		return nil, errors.New("cfg is nil")
	}
	if logger == nil {
		return nil, errors.New("logger is nil")
	}

	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	// example bolt://neo4j:password@localhost:7687
	driver, err := neo4j.NewDriverWithContext(fmt.Sprintf("bolt://%s:%s", cfg.Host, cfg.Port),
		neo4j.BasicAuth(cfg.User, cfg.Passwd, ""), func(config *neo4j.Config) {})
	if err != nil {
		return nil, fmt.Errorf("neo4j new driver: %w", err)
	}

	return driver, nil
}
