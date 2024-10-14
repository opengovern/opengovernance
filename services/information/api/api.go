package api

import (
	"github.com/labstack/echo/v4"
	shared_entities "github.com/opengovern/og-util/pkg/api/shared-entities"
	"github.com/opengovern/opengovernance/services/information/config"
	"github.com/opengovern/opengovernance/services/information/service"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"net/http"
)

type API struct {
	cfg                config.InformationConfig
	tracer             trace.Tracer
	logger             *zap.Logger
	informationService *service.InformationService
}

func New(cfg config.InformationConfig, logger *zap.Logger, informationService *service.InformationService) API {
	return API{
		cfg:                cfg,
		informationService: informationService,
		tracer:             otel.GetTracerProvider().Tracer("information.http.sources"),
		logger:             logger.Named("information-api"),
	}
}

func (s API) Register(e *echo.Echo) {
	g := e.Group("/api/v1/information")
	g.POST("/usage", s.RecordUsage)
}

func (s API) RecordUsage(c echo.Context) error {
	var req shared_entities.CspmUsageRequest
	if err := bindValidate(c, &req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := s.informationService.RecordUsage(c.Request().Context(), req); err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, nil)
}

func bindValidate(ctx echo.Context, i any) error {
	if err := ctx.Bind(i); err != nil {
		return err
	}
	if err := ctx.Validate(i); err != nil {
		return err
	}

	return nil
}
