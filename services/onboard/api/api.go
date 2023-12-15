package api

import "go.uber.org/zap"

type API struct {
	logger *zap.Logger
}

func New(logger *zap.Logger) *API {
	return &API{
		logger: logger.Named("api"),
	}
}
