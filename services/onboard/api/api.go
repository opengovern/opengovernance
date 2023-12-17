package api

import (
	describe "github.com/kaytu-io/kaytu-engine/pkg/describe/client"
	inventory "github.com/kaytu-io/kaytu-engine/pkg/inventory/client"
	"github.com/kaytu-io/kaytu-engine/services/onboard/api/healthz"
	"github.com/kaytu-io/kaytu-engine/services/onboard/db"
	"github.com/kaytu-io/kaytu-engine/services/onboard/meta"
	"github.com/kaytu-io/kaytu-util/pkg/steampipe"
	"github.com/kaytu-io/kaytu-util/pkg/vault"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type API struct {
	logger    *zap.Logger
	describe  describe.SchedulerServiceClient
	inventory inventory.InventoryServiceClient
	meta      *meta.Meta
	steampipe *steampipe.Database
	database  db.Database
	kms       *vault.KMSVaultSourceConfig
}

func New(
	logger *zap.Logger,
	d describe.SchedulerServiceClient,
	i inventory.InventoryServiceClient,
	m *meta.Meta,
	s *steampipe.Database,
	db db.Database,
	kms *vault.KMSVaultSourceConfig,
) *API {
	return &API{
		logger:    logger.Named("api"),
		describe:  d,
		inventory: i,
		meta:      m,
		steampipe: s,
		database:  db,
		kms:       kms,
	}
}

func (*API) Register(e *echo.Echo) {
	var healthz healthz.Healthz

	healthz.Register(e.Group("/healthz"))
}
