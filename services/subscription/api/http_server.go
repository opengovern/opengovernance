package api

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/services/subscription/config"
	"github.com/kaytu-io/kaytu-engine/services/subscription/db"
	"go.uber.org/zap"
)

type HttpServer struct {
	db     db.Database
	logger *zap.Logger
}

func InitializeHttpServer(
	logger *zap.Logger,
	config config.SubscriptionConfig,
) (*HttpServer, error) {
	logger.Info("Initializing http server")

	pdb, err := db.NewDatabase(config.Postgres, logger)
	if err != nil {
		return nil, fmt.Errorf("new postgres client: %w", err)
	}

	return &HttpServer{
		logger: logger,
		db:     pdb,
	}, nil
}
