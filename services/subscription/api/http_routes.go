package api

import (
	api3 "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	"github.com/kaytu-io/kaytu-engine/services/subscription/api/entities"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"net/http"
)

func (h HttpServer) Register(r *echo.Echo) {
	v1 := r.Group("/api/v1")

	v1.GET("/landing", h.HandleLanding)
	v1.GET("/subscriptions", httpserver.AuthorizeHandler(h.ListSubscriptions, api3.KaytuAdminRole))

}

func (h HttpServer) HandleLanding(ctx echo.Context) error {
	token := ctx.QueryParam("token")
	h.logger.Info("landing called", zap.String("token", token))
	return echo.NewHTTPError(http.StatusNotImplemented)
}

func (h HttpServer) ListSubscriptions(ctx echo.Context) error {
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
