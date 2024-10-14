package api

import (
	"github.com/labstack/echo/v4"
	"github.com/opengovern/og-util/pkg/vault"
	describe "github.com/opengovern/opengovernance/pkg/describe/client"
	inventory "github.com/opengovern/opengovernance/pkg/inventory/client"
	"github.com/opengovern/opengovernance/services/integration/api/connection"
	"github.com/opengovern/opengovernance/services/integration/api/connector"
	"github.com/opengovern/opengovernance/services/integration/api/credential"
	"github.com/opengovern/opengovernance/services/integration/api/healthz"
	"github.com/opengovern/opengovernance/services/integration/db"
	"github.com/opengovern/opengovernance/services/integration/meta"
	"github.com/opengovern/opengovernance/services/integration/repository"
	"github.com/opengovern/opengovernance/services/integration/service"
	"go.uber.org/zap"
)

type API struct {
	logger          *zap.Logger
	describe        describe.SchedulerServiceClient
	inventory       inventory.InventoryServiceClient
	meta            *meta.Meta
	database        db.Database
	vault           vault.VaultSourceConfig
	vaultKeyId      string
	masterAccessKey string
	masterSecretKey string
}

func New(
	logger *zap.Logger,
	d describe.SchedulerServiceClient,
	i inventory.InventoryServiceClient,
	m *meta.Meta,
	db db.Database,
	vault vault.VaultSourceConfig,
	vaultKeyId string,
	masterAccessKey string,
	masterSecretKey string,
) *API {
	return &API{
		logger:          logger.Named("api"),
		describe:        d,
		inventory:       i,
		meta:            m,
		database:        db,
		vault:           vault,
		vaultKeyId:      vaultKeyId,
		masterAccessKey: masterAccessKey,
		masterSecretKey: masterSecretKey,
	}
}

func (api *API) Register(e *echo.Echo) {
	var healthz healthz.Healthz

	repo := repository.NewCredConnSQL(api.database)

	connSvc := service.NewConnection(
		repository.NewConnectionSQL(api.database),
		repo,
		api.vault,
		api.vaultKeyId,
		api.describe,
		api.inventory,
		api.meta,
		api.masterAccessKey,
		api.masterSecretKey,
		api.logger,
	)

	credSvc := service.NewCredential(
		repository.NewCredentialSQL(api.database),
		repo,
		api.vault,
		api.vaultKeyId,
		api.describe,
		api.inventory,
		api.meta,
		connSvc,
		api.masterAccessKey,
		api.masterSecretKey,
		api.logger,
	)

	connection := connection.New(
		connSvc,
		credSvc,
		api.logger,
	)

	credential := credential.New(
		credSvc,
		connSvc,
		api.logger,
	)

	connector := connector.New(
		connSvc,
		service.NewConnector(repository.NewConnectorSQL(api.database), api.logger),
		api.logger,
	)

	healthz.Register(e.Group("/api/v1/healthz"))
	connection.Register(e.Group("/api/v1/connections"))
	credential.Register(e.Group("/api/v1/credentials"))
	connector.Register(e.Group("/api/v1/connectors"))
}
