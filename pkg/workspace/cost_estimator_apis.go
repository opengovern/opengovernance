package workspace

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator"
	kaytuResources "github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/resources"

	"github.com/labstack/echo/v4"
	"net/http"
)

// GetAwsCost get azure load balancer cost for a day
// route: /workspace/api/v1/costestimator/aws/{resource_type}
func (s *Server) GetAwsCost(ctx echo.Context) error {
	var request any
	if err := ctx.Bind(&request); err != nil {
		return err
	}
	resourceType := ctx.Param("resource_type")
	s.logger.Info(fmt.Sprintf("calculating cost for %s", resourceType))
	cost, err := costestimator.CalcCosts(s.db, s.logger, "AWS", resourceType,
		kaytuResources.ResourceRequest{Request: request, Address: "test"})
	if err != nil {
		return err
	}
	s.logger.Info(fmt.Sprintf("calculating cost for %s is done, value: %v", resourceType, cost))
	return ctx.JSON(http.StatusOK, cost)
}

// GetAzureCost get azure load balancer cost for a day
// route: /workspace/api/v1/costestimator/azure/{resource_type}
func (s *Server) GetAzureCost(ctx echo.Context) error {
	var request any
	if err := ctx.Bind(&request); err != nil {
		return err
	}
	resourceType := ctx.Param("resource_type")
	s.logger.Info(fmt.Sprintf("calculating cost for %s", resourceType))
	cost, err := costestimator.CalcCosts(s.db, s.logger, "Azure", resourceType,
		kaytuResources.ResourceRequest{Request: request, Address: "test"})
	if err != nil {
		return err
	}
	s.logger.Info(fmt.Sprintf("calculating cost for %s is done, value: %v", resourceType, cost))
	return ctx.JSON(http.StatusOK, cost)
}
