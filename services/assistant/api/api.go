package api

import (
	"github.com/kaytu-io/kaytu-engine/services/assistant/api/thread"
	"github.com/kaytu-io/kaytu-engine/services/assistant/db"
	"github.com/kaytu-io/kaytu-engine/services/assistant/openai"
	"github.com/kaytu-io/kaytu-engine/services/assistant/repository"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type API struct {
	logger   *zap.Logger
	oc       *openai.Service
	database db.Database
}

func New(
	logger *zap.Logger,
	oc *openai.Service,
	database db.Database,
) *API {
	return &API{
		logger:   logger.Named("api"),
		oc:       oc,
		database: database,
	}
}

func (api *API) Register(e *echo.Echo) {
	thr := thread.New(api.logger, api.oc, repository.NewThreadSQL(api.database))
	thr.Register(e.Group("/api/v1/thread"))
}
