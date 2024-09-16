package workspace

import (
	"fmt"
	"github.com/kaytu-io/open-governance/pkg/workspace/api"
	"github.com/kaytu-io/open-governance/pkg/workspace/costestimator"
	kaytuResources "github.com/kaytu-io/open-governance/pkg/workspace/costestimator/resources"

	"github.com/labstack/echo/v4"
	"net/http"
)

// GetAwsCost get azure load balancer cost for a day
// route: /workspace/api/v1/costestimator/aws
func (s *Server) GetAwsCost(ctx echo.Context) error {
	var request api.BaseRequest
	if err := bindValidate(ctx, &request); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := s.CheckRoleInWorkspace(ctx, nil, nil, ""); err != nil {
		return err
	}

	s.logger.Info(fmt.Sprintf("calculating cost for %v", request))
	cost, err := costestimator.CalcCosts(s.db, s.logger, "AWS", request.ResourceType,
		kaytuResources.ResourceRequest{Request: request.Request, Address: request.ResourceId})
	if err != nil {
		return err
	}
	s.logger.Info(fmt.Sprintf("calculating cost for %s is done, value: %v", request.ResourceType, cost))
	return ctx.JSON(http.StatusOK, cost)
}

// GetAzureCost get azure load balancer cost for a day
// route: /workspace/api/v1/costestimator/azure
func (s *Server) GetAzureCost(ctx echo.Context) error {
	var request api.BaseRequest
	if err := bindValidate(ctx, &request); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	if err := s.CheckRoleInWorkspace(ctx, nil, nil, ""); err != nil {
		return err
	}

	s.logger.Info(fmt.Sprintf("calculating cost for %v", request))
	cost, err := costestimator.CalcCosts(s.db, s.logger, "Azure", request.ResourceType,
		kaytuResources.ResourceRequest{Request: request.Request, Address: request.ResourceId})
	if err != nil {
		return err
	}
	s.logger.Info(fmt.Sprintf("calculating cost for %s is done, value: %v", request.ResourceType, cost))
	return ctx.JSON(http.StatusOK, cost)
}

func bindValidate(ctx echo.Context, i interface{}) error {
	if err := ctx.Bind(i); err != nil {
		return err
	}
	if err := ctx.Validate(i); err != nil {
		return err
	}

	return nil
}
