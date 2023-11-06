package cost_estimator

import (
	authapi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"
	"github.com/labstack/echo/v4"
	"net/http"
)

func (h *HttpHandler) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")
	v1.GET("/cost/azure/:resourceId/:resourceType", httpserver.AuthorizeHandler(h.AzureCost, authapi.ViewerRole))
	v1.GET("/cost/aws/:resourceId/:resourceType", httpserver.AuthorizeHandler(h.AwsCost, authapi.ViewerRole))
}

// AzureCost godoc
//
//	@Summary		Get Azure cost
//	@Description	Get Azure cost for each resource
//	@Security		BearerToken
//	@Tags			cost-estimator
//	@Produce		json
//	@Param			resourceId		path		string	true	"ResourceID"
//	@Param			resourceType	path		string	true	"ResourceType"
//	@Success		200				{object}	int
//	@Router			/cost_estimator/api/v1/cost/azure [get]
func (h *HttpHandler) AzureCost(ctx echo.Context) error {
	resourceId := ctx.Param("resourceId")
	resourceType := ctx.Param("resourceType")

	cost, err := azureResourceTypes[resourceType](h, resourceId)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, cost)
}

// AwsCost godoc
//
//	@Summary		Get AWS cost
//	@Description	Get AWS cost for each resource
//	@Security		BearerToken
//	@Tags			cost-estimator
//	@Produce		json
//	@Param			resourceId		path		string	true	"ResourceID"
//	@Param			resourceType	path		string	true	"ResourceType"
//	@Success		200				{object}	int
//	@Router			/cost_estimator/api/v1/cost/aws [get]
func (h *HttpHandler) AwsCost(ctx echo.Context) error {
	resourceId := ctx.Param("resourceId")
	resourceType := ctx.Param("resourceType")

	cost, err := awsResourceTypes[resourceType](h, resourceId)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, cost)
}
