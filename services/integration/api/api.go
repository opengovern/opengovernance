package api

import (
	"github.com/kaytu-io/kaytu-util/pkg/vault"
	describe "github.com/kaytu-io/open-governance/pkg/describe/client"
	inventory "github.com/kaytu-io/open-governance/pkg/inventory/client"
	"github.com/kaytu-io/open-governance/services/integration/api/connection"
	"github.com/kaytu-io/open-governance/services/integration/api/connector"
	"github.com/kaytu-io/open-governance/services/integration/api/credential"
	"github.com/kaytu-io/open-governance/services/integration/api/healthz"
	"github.com/kaytu-io/open-governance/services/integration/db"
	"github.com/kaytu-io/open-governance/services/integration/meta"
	"github.com/kaytu-io/open-governance/services/integration/repository"
	"github.com/kaytu-io/open-governance/services/integration/service"
	"github.com/labstack/echo/v4"
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
