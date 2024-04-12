package api

import (
	"github.com/kaytu-io/kaytu-engine/services/es-sink/api/ingest"
	"github.com/kaytu-io/kaytu-engine/services/es-sink/service"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type API struct {
	logger    *zap.Logger
	ingestApi *ingest.API
}

func New(logger *zap.Logger, esSinkService *service.EsSinkService) *API {
	logger = logger.Named("api-es-sink")
	ingestApi := ingest.New(logger, esSinkService)
	return &API{
		logger:    logger,
		ingestApi: ingestApi,
	}
}

func (api *API) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")

	v1.POST("/ingest", api.ingestApi.Ingest)
}
