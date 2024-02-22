package metadata

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/metadata/internal/database"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"go.uber.org/zap"
)

type HttpHandler struct {
	db     database.Database
	logger *zap.Logger
}

func InitializeHttpHandler(
	postgresUsername string,
	postgresPassword string,
	postgresHost string,
	postgresPort string,
	postgresDb string,
	postgresSSLMode string,
	logger *zap.Logger,
) (*HttpHandler, error) {

	fmt.Println("Initializing http handler")

	cfg := postgres.Config{
		Host:    postgresHost,
		Port:    postgresPort,
		User:    postgresUsername,
		Passwd:  postgresPassword,
		DB:      postgresDb,
		SSLMode: postgresSSLMode,
	}
	orm, err := postgres.NewClient(&cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("new postgres client: %w", err)
	}
	logger.Info("Connected to the postgres database", zap.String("database", postgresDb))

	db := database.NewDatabase(orm)
	err = db.Initialize()
	if err != nil {
		return nil, err
	}
	logger.Info("Initialized database", zap.String("database", postgresDb))

	return &HttpHandler{
		db:     db,
		logger: logger,
	}, nil
}
