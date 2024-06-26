package api

import (
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/alitto/pond"
	"github.com/kaytu-io/kaytu-engine/services/wastage/api/wastage"
	"github.com/kaytu-io/kaytu-engine/services/wastage/config"
	"github.com/kaytu-io/kaytu-engine/services/wastage/cost"
	"github.com/kaytu-io/kaytu-engine/services/wastage/db/repo"
	"github.com/kaytu-io/kaytu-engine/services/wastage/ingestion"
	"github.com/kaytu-io/kaytu-engine/services/wastage/recommendation"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type API struct {
	cfg            config.WastageConfig
	blobClient     *azblob.Client
	blobWorkerPool *pond.WorkerPool
	costSvc        *cost.Service
	recomSvc       *recommendation.Service
	ingestionSvc   *ingestion.Service
	usageRepo      repo.UsageV2Repo
	usageV1Repo    repo.UsageRepo
	userRepo       repo.UserRepo
	orgRepo        repo.OrganizationRepo
	logger         *zap.Logger
}

func New(cfg config.WastageConfig, blobClient *azblob.Client, blobWorkerPool *pond.WorkerPool, costSvc *cost.Service, recomSvc *recommendation.Service, ingestionSvc *ingestion.Service, usageV1Repo repo.UsageRepo, usageRepo repo.UsageV2Repo, userRepo repo.UserRepo, orgRepo repo.OrganizationRepo, logger *zap.Logger) *API {
	return &API{
		cfg:            cfg,
		blobClient:     blobClient,
		blobWorkerPool: blobWorkerPool,
		costSvc:        costSvc,
		recomSvc:       recomSvc,
		ingestionSvc:   ingestionSvc,
		usageV1Repo:    usageV1Repo,
		usageRepo:      usageRepo,
		userRepo:       userRepo,
		orgRepo:        orgRepo,
		logger:         logger.Named("api"),
	}
}

func (api *API) Register(e *echo.Echo) {
	qThr := wastage.New(api.cfg, api.blobClient, api.blobWorkerPool, api.costSvc, api.recomSvc, api.ingestionSvc, api.usageV1Repo, api.usageRepo, api.userRepo, api.orgRepo, api.logger)
	qThr.Register(e)
}
