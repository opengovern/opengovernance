package azure

import (
	"github.com/kaytu-io/kaytu-engine/services/subscription/api/entities"
	"github.com/kaytu-io/kaytu-engine/services/subscription/db"
	"github.com/kaytu-io/kaytu-util/pkg/api"
	"github.com/kaytu-io/kaytu-util/pkg/httpserver"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"net/http"
)

type API struct {
	tracer trace.Tracer
	logger *zap.Logger
	db     db.Database
}

func New(
	logger *zap.Logger,
	db db.Database,
) API {
	return API{
		tracer: otel.GetTracerProvider().Tracer("subscription.http.azure"),
		logger: logger.Named("azure"),
		db:     db,
	}
}

func (h API) HandleLanding(ctx echo.Context) error {
	token := ctx.QueryParam("token")
	h.logger.Info("landing called", zap.String("token", token))
	return echo.NewHTTPError(http.StatusNotImplemented)
}

func (h API) HandleEvent(ctx echo.Context) error {
	h.logger.Info("event called")
	return echo.NewHTTPError(http.StatusNotImplemented)
}

func (h API) ListSubscriptions(ctx echo.Context) error {
	subs, err := h.db.ListSubscriptions()
	if err != nil {
		return err
	}

	var apiSubs []entities.Subscription
	for _, sub := range subs {
		apiSubs = append(apiSubs, entities.Subscription{
			ID:                     sub.ID,
			CreatedAt:              sub.CreatedAt,
			UpdatedAt:              sub.UpdatedAt,
			LifeCycleState:         sub.LifeCycleState,
			ProviderSubscriptionID: sub.ProviderSubscriptionID,
			OwnerResolvingToken:    sub.OwnerResolvingToken,
			OwnerID:                sub.OwnerID,
		})
	}
	return ctx.JSON(http.StatusOK, apiSubs)
}

func (h API) Register(g *echo.Group) {
	g.GET("/landing", h.HandleLanding)
	g.GET("/event", h.HandleEvent)
	g.GET("/subscriptions", httpserver.AuthorizeHandler(h.ListSubscriptions, api.KaytuAdminRole))
}
