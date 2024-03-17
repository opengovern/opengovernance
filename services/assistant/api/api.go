package api

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/services/assistant/api/thread"
	"github.com/kaytu-io/kaytu-engine/services/assistant/db"
	"github.com/kaytu-io/kaytu-engine/services/assistant/openai"
	"github.com/kaytu-io/kaytu-engine/services/assistant/repository"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

type API struct {
	logger               *zap.Logger
	queryAssistant       *openai.Service
	redirectionAssistant *openai.Service
	database             db.Database
}

func New(
	logger *zap.Logger,
	queryAssistant *openai.Service,
	redirectionAssistant *openai.Service,
	database db.Database,
) *API {
	return &API{
		logger:               logger.Named(fmt.Sprintf("api-%s", queryAssistant.AssistantName.String())),
		queryAssistant:       queryAssistant,
		redirectionAssistant: redirectionAssistant,
		database:             database,
	}
}

func (api *API) Register(e *echo.Echo) {
	runRepo := repository.NewRun(api.database)
	qThr := thread.New(api.logger, api.queryAssistant, runRepo)
	qThr.Register(e.Group(fmt.Sprintf("/api/v1/%s/thread", api.queryAssistant.AssistantName.String())))
	rThr := thread.New(api.logger, api.redirectionAssistant, runRepo)
	rThr.Register(e.Group(fmt.Sprintf("/api/v1/%s/thread", api.redirectionAssistant.AssistantName.String())))
}
