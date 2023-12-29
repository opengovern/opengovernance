package metering

import (
	"github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	workspaceClient "github.com/kaytu-io/kaytu-engine/pkg/workspace/client"
	"github.com/kaytu-io/kaytu-engine/services/subscription/api/entities"
	"github.com/kaytu-io/kaytu-engine/services/subscription/db"
	"github.com/kaytu-io/kaytu-engine/services/subscription/service"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"net/http"
	"time"
)

type API struct {
	tracer          trace.Tracer
	logger          *zap.Logger
	db              db.Database
	workspaceClient workspaceClient.WorkspaceServiceClient
	meteringService service.MeteringService
}

func New(
	logger *zap.Logger,
	db db.Database,
	workspaceClient workspaceClient.WorkspaceServiceClient,
	meteringService service.MeteringService,
) API {
	return API{
		tracer:          otel.GetTracerProvider().Tracer("subscription.http.metering"),
		logger:          logger.Named("metering"),
		db:              db,
		workspaceClient: workspaceClient,
		meteringService: meteringService,
	}
}

// GetMeters godoc
//
//	@Summary	Get meters
//	@Security	BearerToken
//	@Tags		subscription
//	@Accept		json
//	@Produce	json
//	@Param		request	body		entities.GetMetersRequest	true	"Request"
//	@Success	200		{object}	entities.GetMetersResponse
//	@Router		/subscription/api/v1/metering/list [get]
func (h API) GetMeters(c echo.Context) error {
	ctx := otel.GetTextMapPropagator().Extract(c.Request().Context(), propagation.HeaderCarrier(c.Request().Header))

	ctx, span := h.tracer.Start(ctx, "get-meters")
	defer span.End()

	var req entities.GetMetersRequest

	if err := c.Bind(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := c.Validate(&req); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	startTime := time.UnixMilli(req.StartTimeEpochMillis)
	endTime := time.UnixMilli(req.EndTimeEpochMillis)
	userID := httpserver.GetUserID(c)

	meters, err := h.meteringService.GetMeters(userID, startTime, endTime)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, entities.GetMetersResponse{
		Meters: meters,
	})
}

func (h API) Register(g *echo.Group) {
	g.GET("/list", httpserver.AuthorizeHandler(h.GetMeters, api.ViewerRole))
}
