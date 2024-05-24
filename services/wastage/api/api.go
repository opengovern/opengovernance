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
	usageV1Repo  repo.UsageRepo
	userRepo     repo.UserRepo
	orgRepo      repo.OrganizationRepo
	logger       *zap.Logger
}

func New(costSvc *cost.Service, recomSvc *recommendation.Service, ingestionSvc *ingestion.Service, usageV1Repo repo.UsageRepo, usageRepo repo.UsageV2Repo, userRepo repo.UserRepo, orgRepo repo.OrganizationRepo, logger *zap.Logger) *API {
	return &API{
		costSvc:      costSvc,
		recomSvc:     recomSvc,
		ingestionSvc: ingestionSvc,
		usageV1Repo:  usageV1Repo,
		usageRepo:    usageRepo,
		userRepo:     userRepo,
		orgRepo:      orgRepo,
		logger:       logger.Named("api"),
	}
}

func (api *API) Register(e *echo.Echo) {
	qThr := wastage.New(api.costSvc, api.recomSvc, api.ingestionSvc, api.usageV1Repo, api.usageRepo, api.userRepo, api.orgRepo, api.logger)
	qThr.Register(e)
}
