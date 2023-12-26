package api

import (
	describe "github.com/kaytu-io/kaytu-engine/pkg/describe/client"
	inventory "github.com/kaytu-io/kaytu-engine/pkg/inventory/client"
	"github.com/kaytu-io/kaytu-engine/services/integration/api/connection"
	"github.com/kaytu-io/kaytu-engine/services/integration/api/credential"
	"github.com/kaytu-io/kaytu-engine/services/integration/api/healthz"
	"github.com/kaytu-io/kaytu-engine/services/integration/db"
	"github.com/kaytu-io/kaytu-engine/services/integration/meta"
	"github.com/kaytu-io/kaytu-engine/services/integration/repository"
	"github.com/kaytu-io/kaytu-engine/services/integration/service"
	"github.com/kaytu-io/kaytu-util/pkg/vault"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type API struct {
	logger          *zap.Logger
	describe        describe.SchedulerServiceClient
	inventory       inventory.InventoryServiceClient
	meta            *meta.Meta
	database        db.Database
	kms             *vault.KMSVaultSourceConfig
	masterAccessKey string
	masterSecretKey string
	arn             string
}

func New(
	logger *zap.Logger,
	d describe.SchedulerServiceClient,
	i inventory.InventoryServiceClient,
	m *meta.Meta,
	db db.Database,
	kms *vault.KMSVaultSourceConfig,
	arn string,
	masterAccessKey string,
	masterSecretKey string,
) *API {
	return &API{
		logger:          logger.Named("api"),
		describe:        d,
		inventory:       i,
		meta:            m,
		database:        db,
		kms:             kms,
		arn:             arn,
		masterAccessKey: masterAccessKey,
		masterSecretKey: masterSecretKey,
	}
}

func (api *API) Register(e *echo.Echo) {
	var healthz healthz.Healthz

	connSvc := service.NewConnection(
		repository.NewConnectionSQL(api.database),
		api.kms,
		api.arn,
		api.describe,
		api.inventory,
		api.meta,
		api.masterAccessKey,
		api.masterSecretKey,
		api.logger,
	)

	connection := connection.New(
		connSvc,
		api.logger,
	)

	credential := credential.New(
		service.NewCredential(
			repository.NewCredentialSQL(api.database),
			api.kms,
			api.arn,
			api.describe,
			api.inventory,
			api.meta,
			connSvc,
			api.masterAccessKey,
			api.masterSecretKey,
			api.logger,
		),
		api.logger,
	)

	healthz.Register(e.Group("/api/v1/healthz"))
	connection.Register(e.Group("/api/v1/connections"))
	credential.Register(e.Group("/api/v1/credentials"))
}
