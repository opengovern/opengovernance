package cost_estimator

import (
	"fmt"
	aws "github.com/kaytu-io/kaytu-aws-describer/aws/model"
	apiAuth "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/cost-estimator/es"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"go.uber.org/zap"
)

func GetEC2InstanceCost(h *HttpHandler, _ string, resourceId string) (float64, error) {
	response, err := es.GetElasticsearch(h.logger, h.client, resourceId, "AWS::EC2::Instance")
	if err != nil {
		h.logger.Error("failed to get resource", zap.Error(err))
		return 0, fmt.Errorf("failed to get resource")
	}
	if len(response.Hits.Hits) == 0 {
		return 0, fmt.Errorf("no resource found")
	}
	var request api.GetEC2InstanceCostRequest
	if instance, ok := response.Hits.Hits[0].Source.Description.(aws.EC2InstanceDescription); ok {
		request = api.GetEC2InstanceCostRequest{
			RegionCode: response.Hits.Hits[0].Source.Location,
			Instance:   instance,
		}
	} else {
		h.logger.Error("cannot parse resource", zap.String("Description",
			fmt.Sprintf("%v", response.Hits.Hits[0].Source.Description)))
		return 0, fmt.Errorf("cannot parse resource")
	}

	cost, err := h.workspaceClient.GetAWS(&httpclient.Context{UserRole: apiAuth.InternalRole}, "aws_instance", request)
	if err != nil {
		h.logger.Error("failed in calculating cost", zap.Error(err))
		return 0, err
	}

	return cost, nil
}

func GetEC2VolumeCost(h *HttpHandler, _ string, resourceId string) (float64, error) {
	response, err := es.GetElasticsearch(h.logger, h.client, resourceId, "AWS::EC2::Volume")
	if err != nil {
		h.logger.Error("failed to get resource", zap.Error(err))
		return 0, fmt.Errorf("failed to get resource")
	}
	if len(response.Hits.Hits) == 0 {
		return 0, fmt.Errorf("no resource found")
	}
	var request api.GetEC2VolumeCostRequest
	if volume, ok := response.Hits.Hits[0].Source.Description.(aws.EC2VolumeDescription); ok {
		request = api.GetEC2VolumeCostRequest{
			RegionCode: response.Hits.Hits[0].Source.Location,
			Volume:     volume,
		}
	} else {
		h.logger.Error("cannot parse resource", zap.String("Description",
			fmt.Sprintf("%v", response.Hits.Hits[0].Source.Description)))
		return 0, fmt.Errorf("cannot parse resource")
	}

	cost, err := h.workspaceClient.GetAWS(&httpclient.Context{UserRole: apiAuth.InternalRole}, "aws_ebs_volume", request)
	if err != nil {
		h.logger.Error("failed in calculating cost", zap.Error(err))
		return 0, err
	}

	return cost, nil
}

func GetELBCost(h *HttpHandler, resourceType string, resourceId string) (float64, error) {
	var response *es.Response
	var err error
	var request api.GetLBCostRequest
	if resourceType == "AWS::ElasticLoadBalancingV2::LoadBalancer" {
		response, err = es.GetElasticsearch(h.logger, h.client, resourceId, "AWS::ElasticLoadBalancingV2::LoadBalancer")
		if err != nil {
			h.logger.Error("failed to get resource", zap.Error(err))
			return 0, fmt.Errorf("failed to get resource")
		}
		if len(response.Hits.Hits) == 0 {
			return 0, fmt.Errorf("no resource found")
		}
		if lb, ok := response.Hits.Hits[0].Source.Description.(aws.ElasticLoadBalancingV2LoadBalancerDescription); ok {
			request = api.GetLBCostRequest{
				RegionCode: response.Hits.Hits[0].Source.Location,
				LBType:     string(lb.LoadBalancer.Type),
			}
		} else {
			h.logger.Error("cannot parse resource", zap.String("Description",
				fmt.Sprintf("%v", response.Hits.Hits[0].Source.Description)))
			return 0, fmt.Errorf("cannot parse resource")
		}
	} else if resourceType == "AWS::ElasticLoadBalancing::LoadBalancer" {
		response, err = es.GetElasticsearch(h.logger, h.client, resourceId, "AWS::ElasticLoadBalancing::LoadBalancer")
		if err != nil {
			h.logger.Error("failed to get resource", zap.Error(err))
			return 0, fmt.Errorf("failed to get resource")
		}
		if len(response.Hits.Hits) == 0 {
			return 0, fmt.Errorf("no resource found")
		}
		if _, ok := response.Hits.Hits[0].Source.Description.(aws.ElasticLoadBalancingLoadBalancerDescription); ok {
			request = api.GetLBCostRequest{
				RegionCode: response.Hits.Hits[0].Source.Location,
				LBType:     "classic",
			}
		} else {
			return 0, fmt.Errorf("cannot parse resource")
		}
	}

	cost, err := h.workspaceClient.GetAWS(&httpclient.Context{UserRole: apiAuth.InternalRole}, "aws_elb", request)
	if err != nil {
		h.logger.Error("failed in calculating cost", zap.Error(err))
		return 0, err
	}

	return cost, nil
}

func GetRDSInstanceCost(h *HttpHandler, _ string, resourceId string) (float64, error) {
	response, err := es.GetElasticsearch(h.logger, h.client, resourceId, "AWS::RDS::DBInstance")
	if err != nil {
		h.logger.Error("failed to get resource", zap.Error(err))
		return 0, fmt.Errorf("failed to get resource")
	}
	if len(response.Hits.Hits) == 0 {
		return 0, fmt.Errorf("no resource found")
	}
	var request api.GetRDSInstanceRequest
	if dbInstance, ok := response.Hits.Hits[0].Source.Description.(aws.RDSDBInstanceDescription); ok {
		request = api.GetRDSInstanceRequest{
			RegionCode: response.Hits.Hits[0].Source.Location,
			DBInstance: dbInstance,
		}
	} else {
		h.logger.Error("cannot parse resource", zap.String("Description",
			fmt.Sprintf("%v", response.Hits.Hits[0].Source.Description)))
		return 0, fmt.Errorf("cannot parse resource")
	}
	cost, err := h.workspaceClient.GetAWS(&httpclient.Context{UserRole: apiAuth.InternalRole}, "aws_db_instance", request)
	if err != nil {
		h.logger.Error("failed in calculating cost", zap.Error(err))
		return 0, err
	}

	return cost, nil
}
