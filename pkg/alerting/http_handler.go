package alerting

import (
	"fmt"
	"github.com/kaytu-io/kaytu-util/pkg/postgres"
	"go.uber.org/zap"
)

type HttpHandler struct {
	db     Database
	logger *zap.Logger
}

func InitializeHttpHandler(
	postgresHost string, postgresPort string, postgresDb string, postgresUsername string, postgresPassword string, postgresSSLMode string,
	logger *zap.Logger,
) (h *HttpHandler, err error) {
	h = &HttpHandler{}

	fmt.Println("Initializing http handler")

	// setup postgres connection
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

	h.db = Database{orm: orm}
	fmt.Println("Connected to the postgres database: ", postgresDb)

	err = h.db.Initialize()
	if err != nil {
		return nil, err
	}
	fmt.Println("Initialized postgres database: ", postgresDb)

	return h, nil
}
