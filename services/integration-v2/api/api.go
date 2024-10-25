package api

import (
	"github.com/labstack/echo/v4"
	"github.com/opengovern/og-util/pkg/vault"
	"github.com/opengovern/opengovernance/services/integration-v2/api/credentials"
	"github.com/opengovern/opengovernance/services/integration-v2/api/integrations"
	"github.com/opengovern/opengovernance/services/integration-v2/db"
	"go.uber.org/zap"
)

type API struct {
	logger          *zap.Logger
	database        db.Database
	vault           vault.VaultSourceConfig
	vaultKeyId      string
	masterAccessKey string
	masterSecretKey string
}

func New(
	logger *zap.Logger,
	db db.Database,
	vault vault.VaultSourceConfig,
) *API {
	return &API{
		logger:   logger.Named("api"),
		database: db,
		vault:    vault,
	}
}

func (api *API) Register(e *echo.Echo) {
	integrationsApi := integrations.New(api.vault, api.database, api.logger)
	cred := credentials.New(api.database, api.logger)

	integrationsApi.Register(e.Group("/api/v1/integrations"))
	cred.Register(e.Group("/api/v1/credentials"))
}
