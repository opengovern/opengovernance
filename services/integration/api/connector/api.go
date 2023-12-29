package connector

import (
	"encoding/json"
	"net/http"

	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	"github.com/kaytu-io/kaytu-engine/pkg/onboard/api"
	"github.com/kaytu-io/kaytu-engine/services/integration/api/entity"
	"github.com/kaytu-io/kaytu-engine/services/integration/model"
	"github.com/kaytu-io/kaytu-engine/services/integration/service"
	"github.com/kaytu-io/kaytu-util/pkg/source"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

type API struct {
	connSvc service.Connection
	credSvc service.Credential
	tracer  trace.Tracer
	logger  *zap.Logger
}

func New(
	connSvc service.Connection,
	credSvc service.Credential,
	logger *zap.Logger,
) API {
	return API{
		connSvc: connSvc,
		credSvc: credSvc,
		tracer:  otel.GetTracerProvider().Tracer("integration.http.connector"),
		logger:  logger.Named("source"),
	}
}

// List godoc
//
//	@Summary		List connectors
//	@Description	Returns list of all connectors
//	@Security		BearerToken
//	@Tags			connectors
//	@Produce		json
//	@Success		200	{object}	[]entity.ConnectorCount
//	@Router			/integration/api/v1/connector [get]
func (h API) List(c echo.Context) error {
	// trace :
	ctx, span := h.tracer.Start(c.Request().Context(), "new_ListConnectors", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	connectors, err := h.db.ListConnectors()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	var res []entity.ConnectorCount

	for _, c := range connectors {
		_, span3 := tracer.Start(outputS2, "new_CountSourcesOfType", trace.WithSpanKind(trace.SpanKindServer))
		span3.SetName("new_CountSourcesOfType")

		count, err := h.db.CountSourcesOfType(c.Name)
		if err != nil {
			span3.RecordError(err)
			span3.SetStatus(codes.Error, err.Error())
			return err
		}
		span3.AddEvent("information", trace.WithAttributes(
			attribute.String("source name", string(c.Name)),
		))
		span3.End()

		tags := make(map[string]any)
		err = json.Unmarshal(c.Tags, &tags)
		if err != nil {
			return err
		}
		res = append(res, entity.ConnectorCount{
			Connector: entity.Connector{
				Name:                c.Name,
				Label:               c.Label,
				ShortDescription:    c.ShortDescription,
				Description:         c.Description,
				Direction:           c.Direction,
				Status:              c.Status,
				Logo:                c.Logo,
				AutoOnboardSupport:  c.AutoOnboardSupport,
				AllowNewConnections: c.AllowNewConnections,
				MaxConnectionLimit:  c.MaxConnectionLimit,
				Tags:                tags,
			},
			ConnectionCount: count,
		})
	}
	span2.End()

	return c.JSON(http.StatusOK, res)
}

// CatalogMetrics godoc
//
//	@Summary		List catalog metrics
//	@Description	Retrieving the list of metrics for catalog page.
//	@Security		BearerToken
//	@Tags			onboard
//	@Produce		json
//	@Param			connector	query		[]source.Type	false	"Connector"
//	@Success		200			{object}	api.CatalogMetrics
//	@Router			/onboard/api/v1/catalog/metrics [get]
func (h API) CatalogMetrics(c echo.Context) error {
	var metrics api.CatalogMetrics
	// trace :
	ctx, span := h.tracer.Start(ctx.Request().Context(), "new_ListSources", trace.WithSpanKind(trace.SpanKindServer))
	span.SetName("new_ListSources")

	connectors := source.ParseTypes(httpserver.QueryArrayParam(ctx, "connector"))

	srcs, err := h.connSvc.ListWithFilter(ctx, connectors, nil, nil, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return echo.NewHTTPError(http.StatusInternalServerError)
	}
	span.End()

	for _, src := range srcs {
		metrics.TotalConnections++
		if src.LifecycleState.IsEnabled() {
			metrics.ConnectionsEnabled++
		}

		switch src.HealthState {
		case source.HealthStatusHealthy:
			metrics.HealthyConnections++
		case source.HealthStatusUnhealthy:
			metrics.UnhealthyConnections++
		}

		if src.LifecycleState == model.ConnectionLifecycleStateInProgress {
			metrics.InProgressConnections++
		}
	}

	return ctx.JSON(http.StatusOK, metrics)
}
