package api

import (
	"github.com/labstack/echo/v4"
	"github.com/opengovern/og-util/pkg/vault"
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
	vaultKeyId string,
	masterAccessKey string,
	masterSecretKey string,
) *API {
	return &API{
		logger:          logger.Named("api"),
		database:        db,
		vault:           vault,
		vaultKeyId:      vaultKeyId,
		masterAccessKey: masterAccessKey,
		masterSecretKey: masterSecretKey,
	}
}

func (api *API) Register(e *echo.Echo) {

}
