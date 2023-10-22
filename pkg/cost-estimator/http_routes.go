package cost_estimator

import (
	authapi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"
	"github.com/labstack/echo/v4"
	"net/http"
)

func (h *HttpHandler) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")
	v1.GET("/cost/azure", httpserver.AuthorizeHandler(h.AzureCost, authapi.ViewerRole))
	v1.PUT("/table/store/azure", httpserver.AuthorizeHandler(h.TriggerStoreAzureCostTable, authapi.AdminRole))
}

// AzureCost godoc
//
//	@Summary		Get Azure cost
//	@Description	Get Azure cost for each resource
//	@Security		BearerToken
//	@Tags			cost-estimator
//	@Produce		int
//	@Param			resourceId	query		string	true	"ResourceID"
//	@Param			resourceType	query		string	true	"ResourceType"
//	@Success		200		{object}
//	@Router			/cost-estimator/api/v1/cost/azure [get]
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
//	@Produce		int
//	@Param			resourceId	path		string	true	"ResourceID"
//	@Param			resourceType	path		string	true	"ResourceType"
//	@Success		200		{object}
//	@Router			/cost-estimator/api/v1/cost/aws [get]
func (h *HttpHandler) AwsCost(ctx echo.Context) error {
	resourceId := ctx.Param("resourceId")
	resourceType := ctx.Param("resourceType")

	cost, err := awsResourceTypes[resourceType](h, resourceId)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, cost)
}

// TriggerStoreAzureCostTable godoc
//
//	@Summary		Trigger Azure Cost Table Store
//	@Description	Trigger azure cost table store
//	@Security		BearerToken
//	@Tags			cost-estimator
//	@Produce		int
//	@Success		200		{object}
//	@Router			/cost-estimator/api/v1/cost/aws [get]
func (h *HttpHandler) TriggerStoreAzureCostTable(ctx echo.Context) error {
	err := h.HandleStoreAzureCostTable()
	if err != nil {
		return ctx.String(http.StatusInternalServerError, err.Error())
	}

	return ctx.NoContent(http.StatusOK)
}
