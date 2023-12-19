package connection

import (
	"net/http"

	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	"github.com/kaytu-io/kaytu-engine/services/integration/api/entity"
	"github.com/kaytu-io/kaytu-engine/services/integration/service"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type API struct {
	svc    service.Connection
	tracer trace.Tracer
	logger *zap.Logger
}

func New(
	svc service.Connection,
	logger *zap.Logger,
) API {
	return API{
		svc:    svc,
		tracer: otel.GetTracerProvider().Tracer("integration.http.sources"),
		logger: logger.Named("source"),
	}
}

func (h API) List(c echo.Context) error {
	ctx := otel.GetTextMapPropagator().Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))

	ctx, span := h.tracer.Start(ctx, "list")
	defer span.End()

	types := httpserver.QueryArrayParam(c, "connector")

	sources, err := h.svc.List(ctx, source.ParseTypes(types))
	if err != nil {
		h.logger.Error("failed to read sources from the service", zap.Error(err))

		return echo.ErrInternalServerError
	}

	var resp entity.ListConnectionsResponse
	for _, s := range sources {
		apiRes := entity.NewConnection(s)
		if httpserver.GetUserRole(c) == api.InternalRole {
			apiRes.Credential = entity.NewCredential(s.Credential)
			apiRes.Credential.Config = s.Credential.Secret
			if apiRes.Credential.Version == 2 {
				apiRes.Credential.Config, err = h.svc.CredentialV2ToV1(s.Credential.Secret)
				if err != nil {
					h.logger.Error("failed to provide credential from v2 to v1", zap.Error(err))

					return echo.ErrInternalServerError
				}
			}
		}
		resp = append(resp, apiRes)
	}

	return c.JSON(http.StatusOK, resp)
}

func (h API) Get(c echo.Context) error {
	ctx := otel.GetTextMapPropagator().Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))

	ctx, span := h.tracer.Start(ctx, "get")
	defer span.End()

	var req entity.GetConnectionsRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	conns, err := h.svc.Get(ctx, req.SourceIDs)
	if err != nil {
		h.logger.Error("failed to read connections from the service", zap.Error(err))

		return echo.ErrInternalServerError
	}

	var res []entity.Connection
	for _, conn := range conns {
		apiRes := entity.NewConnection(conn)
		if httpserver.GetUserRole(c) == api.InternalRole {
			apiRes.Credential = entity.NewCredential(conn.Credential)
			apiRes.Credential.Config = conn.Credential.Secret
			if apiRes.Credential.Version == 2 {
				apiRes.Credential.Config, err = h.svc.CredentialV2ToV1(conn.Credential.Secret)
				if err != nil {
					return err
				}
			}

		}

		res = append(res, apiRes)
	}
	return c.JSON(http.StatusOK, res)
}

func (h API) Count(c echo.Context) error {
	ctx := otel.GetTextMapPropagator().Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))

	ctx, span := h.tracer.Start(ctx, "count")
	defer span.End()

	sType := c.QueryParam("connector")

	var st *source.Type

	if sType != "" {
		t, err := source.ParseType(sType)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, err.Error())
		}
		st = &t
	}

	count, err := h.svc.Count(ctx, st)
	if err != nil {
		h.logger.Error("failed to read connections from the service", zap.Error(err))

		return echo.ErrInternalServerError
	}

	return c.JSON(http.StatusOK, count)
}

func (s API) Register(g *echo.Group) {
	g.GET("/", httpserver.AuthorizeHandler(s.List, api.ViewerRole))
	g.POST("/", httpserver.AuthorizeHandler(s.Get, api.KaytuAdminRole))
	g.GET("/count", httpserver.AuthorizeHandler(s.Count, api.ViewerRole))
}
