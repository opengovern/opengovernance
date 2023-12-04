package cost_estimator

import (
	"fmt"
	authapi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/httpserver"
	"github.com/labstack/echo/v4"
	"net/http"
)

func (h *HttpHandler) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")
	v1.GET("/cost/azure", httpserver.AuthorizeHandler(h.AzureCost, authapi.ViewerRole))
	v1.GET("/cost/aws", httpserver.AuthorizeHandler(h.AwsCost, authapi.ViewerRole))
}

// AzureCost godoc
//
//	@Summary		Get Azure cost
//	@Description	Get Azure cost for each resource
//	@Security		BearerToken
//	@Tags			cost-estimator
//	@Produce		json
//	@Param			resourceId		query		string	true	"Connection ID"
//	@Param			resourceType	query		string	true	"ResourceType"
//	@Success		200				{object}	int
//	@Router			/cost_estimator/api/v1/cost/azure [get]
func (h *HttpHandler) AzureCost(ctx echo.Context) error {
	resourceId := ctx.QueryParam("resourceId")
	resourceType := ctx.QueryParam("resourceType")

	if _, ok := azureResourceTypes[resourceType]; !ok {
		return ctx.JSON(http.StatusBadRequest, fmt.Errorf("resource type not found"))
	}
	cost, err := azureResourceTypes[resourceType](h, resourceType, resourceId)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, err.Error())
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
//	@Param			resourceId		query		string	true	"Connection ID"
//	@Param			resourceType	query		string	true	"ResourceType"
//	@Success		200				{object}	int
//	@Router			/cost_estimator/api/v1/cost/aws [get]
func (h *HttpHandler) AwsCost(ctx echo.Context) error {
	resourceId := ctx.QueryParam("resourceId")
	resourceType := ctx.QueryParam("resourceType")

	if _, ok := awsResourceTypes[resourceType]; !ok {
		return ctx.JSON(http.StatusBadRequest, fmt.Errorf("resource type not found"))
	}
	cost, err := awsResourceTypes[resourceType](h, resourceType, resourceId)
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, err.Error())
	}

	return ctx.JSON(http.StatusOK, cost)
}
