package workspace

import (
	"github.com/kaytu-io/kaytu-engine/pkg/cost-estimator/es"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/aws"
	"github.com/labstack/echo/v4"
	"net/http"
)

// GetEC2InstanceCost Calculates ec2 instance price for a day
// route: /workspace/api/v1/cost_estimator/ec2instance [get]
func (s *Server) GetEC2InstanceCost(ctx echo.Context) error {
	var request es.EC2InstanceResponse
	if err := ctx.Bind(&request); err != nil {
		return err
	}

	cost, err := aws.EC2InstanceCostByResource(s.costEstimatorDb, request)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, cost)
}

// GetEC2VolumeCost Calculates ec2 volume (ebs volume) price for a day
// route: /workspace/api/v1/cost_estimator/ec2volume [get]
func (s *Server) GetEC2VolumeCost(ctx echo.Context) error {
	var request es.EC2VolumeResponse
	if err := ctx.Bind(&request); err != nil {
		return err
	}

	cost, err := aws.EC2VolumeCostByResource(s.costEstimatorDb, request)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, cost)
}

// GetLBCost Calculates load balancers price for a day
// route: /workspace/api/v1/cost_estimator/loadbalancer [get]
func (s *Server) GetLBCost(ctx echo.Context) error {
	var request es.LBRequest
	if err := ctx.Bind(&request); err != nil {
		return err
	}

	cost, err := aws.LBCostByResource(s.costEstimatorDb, request)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, cost)
}
