package api

import (
	workspaceClient "github.com/kaytu-io/kaytu-engine/pkg/workspace/client"
	"github.com/kaytu-io/kaytu-engine/services/subscription/api/azure"
	"github.com/kaytu-io/kaytu-engine/services/subscription/api/metering"
	"github.com/kaytu-io/kaytu-engine/services/subscription/db"
	"github.com/kaytu-io/kaytu-engine/services/subscription/service"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type API struct {
	logger          *zap.Logger
	database        db.Database
	workspaceClient workspaceClient.WorkspaceServiceClient
	meteringService service.MeteringService
}

func New(
	logger *zap.Logger,
	db db.Database,
	workspaceClient workspaceClient.WorkspaceServiceClient,
	meteringService service.MeteringService,
) *API {
	return &API{
		logger:          logger.Named("api"),
		database:        db,
		workspaceClient: workspaceClient,
		meteringService: meteringService,
	}
}

func (api *API) Register(e *echo.Echo) {
	azure := azure.New(
		api.logger,
		api.database,
	)

	metering := metering.New(
		api.logger,
		api.database,
		api.workspaceClient,
		api.meteringService,
	)

	azure.Register(e.Group("/api/v1/marketplace/azure"))
	metering.Register(e.Group("/api/v1/metering"))
}
