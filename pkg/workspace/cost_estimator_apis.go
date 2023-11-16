package workspace

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/aws"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/azure"
	kaytuResources "github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/resources"

	"github.com/labstack/echo/v4"
	"net/http"
)

// GetEC2InstanceCost Calculates ec2 instance cost for a day
// route: /workspace/api/v1/costestimator/aws/ec2instance [get]
func (s *Server) GetEC2InstanceCost(ctx echo.Context) error {
	var request api.GetEC2InstanceCostRequest
	if err := ctx.Bind(&request); err != nil {
		return err
	}

	cost, err := aws.EC2InstanceCostByResource(s.db, request)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, cost)
}

// GetEC2VolumeCost Calculates ec2 volume (ebs volume) cost for a day
// route: /workspace/api/v1/costestimator/aws/ec2volume [get]
func (s *Server) GetEC2VolumeCost(ctx echo.Context) error {
	var request api.GetEC2VolumeCostRequest
	if err := ctx.Bind(&request); err != nil {
		return err
	}

	cost, err := aws.EC2VolumeCostByResource(s.db, request)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, cost)
}

// GetLBCost Calculates load balancers cost for a day
// route: /workspace/api/v1/costestimator/aws/loadbalancer [get]
func (s *Server) GetLBCost(ctx echo.Context) error {
	var request api.GetLBCostRequest
	if err := ctx.Bind(&request); err != nil {
		return err
	}

	cost, err := aws.LBCostByResource(s.db, request)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, cost)
}

// GetRDSInstanceCost get rds instance cost for a day
// route: /workspace/api/v1/costestimator/aws/rdsinstance [get]
func (s *Server) GetRDSInstanceCost(ctx echo.Context) error {
	var request api.GetRDSInstanceRequest
	if err := ctx.Bind(&request); err != nil {
		return err
	}

	cost, err := aws.RDSDBInstanceCostByResource(s.db, request)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, cost)
}

// GetAzureVmCost get azure virtual machine cost for a day
// route: /workspace/api/v1/costestimator/azure/virtualmachine
func (s *Server) GetAzureVmCost(ctx echo.Context) error {
	var request api.GetAzureVmRequest
	if err := ctx.Bind(&request); err != nil {
		return err
	}

	cost, err := azure.VmCostByResource(s.db, request)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, cost)
}

// GetAzureManagedStorageCost get azure managed storage cost for a day
// route: /workspace/api/v1/costestimator/azure/managedstorage
func (s *Server) GetAzureManagedStorageCost(ctx echo.Context) error {
	var request api.GetAzureManagedStorageRequest
	if err := ctx.Bind(&request); err != nil {
		return err
	}

	cost, err := azure.ManagedStorageCostByResource(s.db, request)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, cost)
}

// GetAzureLoadBalancerCost get azure load balancer cost for a day
// route: /workspace/api/v1/costestimator/azure/loadbalancer
func (s *Server) GetAzureLoadBalancerCost(ctx echo.Context) error {
	var request api.GetAzureLoadBalancerRequest
	if err := ctx.Bind(&request); err != nil {
		return err
	}

	cost, err := azure.LbCostByResource(s.db, request)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, cost)
}

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
