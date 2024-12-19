package api

import (
	"github.com/labstack/echo/v4"
	"github.com/opengovern/opencomply/services/rego/api/rego"
	"github.com/opengovern/opencomply/services/rego/internal"
	"go.uber.org/zap"
)

type API struct {
	logger  *zap.Logger
	Service *internal.RegoEngine
}

func New(logger *zap.Logger, service *internal.RegoEngine) *API {
	return &API{
		logger:  logger.Named("api"),
		Service: service,
	}
}

func (api *API) Register(e *echo.Echo) {
	evaluateApi := rego.New(api.logger, api.Service)
	evaluateApi.Register(e.Group("/api/v1/rego"))
}
