package api

import (
	"github.com/kaytu-io/kaytu-engine/services/wastage/api/wastage"
	"github.com/kaytu-io/kaytu-engine/services/wastage/cost"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type API struct {
	costSvc *cost.Service
	logger  *zap.Logger
}

func New(
	costSvc *cost.Service,
	logger *zap.Logger,
) *API {
	return &API{
		costSvc: costSvc,
		logger:  logger.Named("api"),
	}
}

func (api *API) Register(e *echo.Echo) {
	qThr := wastage.New(api.costSvc, api.logger)
	qThr.Register(e.Group("/api/v1/wastage"))
}
