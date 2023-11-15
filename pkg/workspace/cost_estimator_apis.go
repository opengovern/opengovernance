package workspace

import (
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/aws"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/azure"
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

// GetAzureSqlServerDatabase get azure RDS Instance cost for a day
// route: /workspace/api/v1/costestimator/azure/sqlserverdatabase
func (s *Server) GetAzureSqlServerDatabase(ctx echo.Context) error {
	var request api.GetAzureSqlServersDatabasesRequest
	if err := ctx.Bind(&request); err != nil {
		return err
	}

	cost, err := azure.SqlServerDatabaseCostByResource(s.db, request, s.logger)
	if err != nil {
		return err
	}
	return ctx.JSON(http.StatusOK, cost)
}
