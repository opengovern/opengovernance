package cost_estimator

import (
	authapi "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	cost_calculator "github.com/kaytu-io/kaytu-engine/pkg/cost-estimator/cost-calculator"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpserver"
	"github.com/labstack/echo/v4"
	"net/http"
)

func (h HttpHandler) Register(e *echo.Echo) {
	v1 := e.Group("/api/v1")
	v1.GET("/cost/azure/:resourceId", httpserver.AuthorizeHandler(h.AzureCost, authapi.ViewerRole))
}

// AzureCost godoc
//
//	@Summary		Get Azure cost
//	@Description	Get Azure cost for each resource
//	@Security		BearerToken
//	@Tags			cost-estimator
//	@Produce		int
//	@Param			resourceId	path		string	true	"ResourceID"
//	@Success		200		{object}
//	@Router			/cost_estimator/api/v1/cost/azure [get]
func (h *HttpHandler) AzureCost(ctx echo.Context) error {
	resourceId := ctx.Param("resourceId")
	resource, err := GetAzureResource(h, resourceId)
	if err != nil {
		return err
	}

	OSType := resource.Description.VirtualMachine.Properties.StorageProfile.OSDisk.OSType
	location := resource.Description.VirtualMachine.Location
	VMSize := resource.Description.VirtualMachine.Properties.HardwareProfile.VMSize
	cost, err := cost_calculator.AzureCostEstimator(OSType, location, VMSize)
	if err != nil {
		return err
	}

	return ctx.JSON(http.StatusOK, cost)
}
