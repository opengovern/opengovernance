package api

import (
	"github.com/labstack/echo/v4"
	"github.com/opengovern/og-util/pkg/steampipe"
	"github.com/opengovern/og-util/pkg/vault"
	"github.com/opengovern/opengovernance/services/integration/api/credentials"
	"github.com/opengovern/opengovernance/services/integration/api/integrations"
	"github.com/opengovern/opengovernance/services/integration/db"
	"go.uber.org/zap"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type API struct {
	logger          *zap.Logger
	database        db.Database
	steampipeConn   *steampipe.Database
	vault           vault.VaultSourceConfig
	kubeClient      client.Client
	vaultKeyId      string
	masterAccessKey string
	masterSecretKey string
}

func New(
	logger *zap.Logger,
	db db.Database,
	vault vault.VaultSourceConfig,
	steampipeConn *steampipe.Database,
	kubeClient client.Client,
) *API {
	return &API{
		logger:        logger.Named("api"),
		database:      db,
		vault:         vault,
		steampipeConn: steampipeConn,
		kubeClient:    kubeClient,
	}
}

func (api *API) Register(e *echo.Echo) {
	integrationsApi := integrations.New(api.vault, api.database, api.logger, api.steampipeConn, api.kubeClient)
	cred := credentials.New(api.vault, api.database, api.logger)

	integrationsApi.Register(e.Group("/api/v1/integrations"))
	cred.Register(e.Group("/api/v1/credentials"))
}
