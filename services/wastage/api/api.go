package api

import (
	"github.com/kaytu-io/kaytu-engine/services/wastage/api/wastage"
	"github.com/kaytu-io/kaytu-engine/services/wastage/cost"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/repo"
	"github.com/kaytu-io/kaytu-engine/services/wastage/ingestion"
	"github.com/kaytu-io/kaytu-engine/services/wastage/recommendation"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type API struct {
	costSvc      *cost.Service
	recomSvc     *recommendation.Service
	ingestionSvc *ingestion.Service
	usageRepo    repo.UsageV2Repo
	logger       *zap.Logger
}

func New(costSvc *cost.Service, recomSvc *recommendation.Service, ingestionSvc *ingestion.Service, usageRepo repo.UsageV2Repo, logger *zap.Logger) *API {
	return &API{
		costSvc:      costSvc,
		recomSvc:     recomSvc,
		ingestionSvc: ingestionSvc,
		usageRepo:    usageRepo,
		logger:       logger.Named("api"),
	}
}

func (api *API) Register(e *echo.Echo) {
	qThr := wastage.New(api.costSvc, api.recomSvc, api.ingestionSvc, api.usageRepo, api.logger)
	qThr.Register(e)
}
