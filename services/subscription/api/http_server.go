package api

import (
	"github.com/kaytu-io/kaytu-engine/services/subscription/config"
	"github.com/kaytu-io/kaytu-engine/services/subscription/db"
	"go.uber.org/zap"
)

type HttpServer struct {
	db     db.Database
	logger *zap.Logger
}

func InitializeHttpServer(logger *zap.Logger, config config.SubscriptionConfig, pdb db.Database) (*HttpServer, error) {
	logger.Info("Initializing http server")

	return &HttpServer{
		logger: logger,
		db:     pdb,
	}, nil
}
