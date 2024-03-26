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
	logger              *zap.Logger
	queryAssistant      *openai.Service
	assetsAssistant     *openai.Service
	scoreAssistant      *openai.Service
	complianceAssistant *openai.Service
	database            db.Database
}

func New(
	logger *zap.Logger,
	queryAssistant *openai.Service,
	assetsAssistant *openai.Service,
	scoreAssistant *openai.Service,
	complianceAssistant *openai.Service,
	database db.Database,
) *API {
	return &API{
		logger:              logger.Named(fmt.Sprintf("api-%s", queryAssistant.AssistantName.String())),
		queryAssistant:      queryAssistant,
		assetsAssistant:     assetsAssistant,
		scoreAssistant:      scoreAssistant,
		complianceAssistant: complianceAssistant,
		database:            database,
	}
}

func (api *API) Register(e *echo.Echo) {
	runRepo := repository.NewRun(api.database)
	qThr := thread.New(api.logger, api.queryAssistant, runRepo)
	qThr.Register(e.Group(fmt.Sprintf("/api/v1/%s/thread", api.queryAssistant.AssistantName.String())))
	rThr := thread.New(api.logger, api.assetsAssistant, runRepo)
	rThr.Register(e.Group(fmt.Sprintf("/api/v1/%s/thread", api.assetsAssistant.AssistantName.String())))
	sThr := thread.New(api.logger, api.scoreAssistant, runRepo)
	sThr.Register(e.Group(fmt.Sprintf("/api/v1/%s/thread", api.scoreAssistant.AssistantName.String())))
	cThr := thread.New(api.logger, api.complianceAssistant, runRepo)
	cThr.Register(e.Group(fmt.Sprintf("/api/v1/%s/thread", api.complianceAssistant.AssistantName.String())))
}
