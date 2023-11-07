package workspace

import (
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/aws"
	"github.com/labstack/echo/v4"
	"net/http"
)

// GetEC2InstanceCost Calculates ec2 instance price for a day
// route: /workspace/api/v1/costestimator/ec2instance [get]
func (s *Server) GetEC2InstanceCost(ctx echo.Context) error {
	var request api.GetEC2InstanceCostRequest
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
// route: /workspace/api/v1/costestimator/ec2volume [get]
func (s *Server) GetEC2VolumeCost(ctx echo.Context) error {
	var request api.GetEC2VolumeCostRequest
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
// route: /workspace/api/v1/costestimator/loadbalancer [get]
func (s *Server) GetLBCost(ctx echo.Context) error {
	var request api.GetLBCostRequest
	if err := ctx.Bind(&request); err != nil {
		return err
	}

	cost, err := aws.LBCostByResource(s.costEstimatorDb, request)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, cost)
}

// GetRDSInstanceCost get rds instance price from database
// route: /workspace/api/v1/costestimator/rdsinstance [get]
func (s *Server) GetRDSInstanceCost(ctx echo.Context) error {
	var request api.GetRDSInstanceRequest
	if err := ctx.Bind(&request); err != nil {
		return err
	}

	cost, err := aws.RDSDBInstanceCostByResource(s.costEstimatorDb, request)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, cost)
}
