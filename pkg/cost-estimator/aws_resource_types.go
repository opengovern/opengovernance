package cost_estimator

import (
	"fmt"
	aws "github.com/kaytu-io/kaytu-aws-describer/aws/model"
	apiAuth "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/cost-estimator/es"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/mitchellh/mapstructure"
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
	if description, ok := response.Hits.Hits[0].Source.Description.(map[string]interface{}); ok {
		instanceInterface, instanceExists := description["Instance"]
		if !instanceExists {
			h.logger.Error("cannot find 'Instance' field in Description", zap.Any("Description", response.Hits.Hits[0].Source.Description))
			return 0, fmt.Errorf("cannot find 'Instance' field in Description")
		}

		var instanceStruct aws.EC2InstanceDescription
		decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			Result:  &instanceStruct,
			TagName: "json",
		})
		if err != nil {
			h.logger.Error("error creating mapstructure decoder", zap.Error(err))
			return 0, err
		}

		if err := decoder.Decode(instanceInterface); err != nil {
			h.logger.Error("error decoding 'Instance' field", zap.Error(err))
			return 0, err
		}

		request = api.GetEC2InstanceCostRequest{
			RegionCode: response.Hits.Hits[0].Source.Location,
			Instance:   instanceStruct,
		}
	} else {
		h.logger.Error("cannot parse resource", zap.String("Description", fmt.Sprintf("%v", response.Hits.Hits[0].Source.Description)))
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
	if description, ok := response.Hits.Hits[0].Source.Description.(map[string]interface{}); ok {
		volumeInterface, volumeExists := description["Volume"]
		if !volumeExists {
			h.logger.Error("cannot find 'Volume' field in Description", zap.Any("Description", response.Hits.Hits[0].Source.Description))
			return 0, fmt.Errorf("cannot find 'Volume' field in Description")
		}

		var volumeStruct aws.EC2VolumeDescription
		decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			Result:  &volumeStruct,
			TagName: "json",
		})
		if err != nil {
			h.logger.Error("error creating mapstructure decoder", zap.Error(err))
			return 0, err
		}

		if err := decoder.Decode(volumeInterface); err != nil {
			h.logger.Error("error decoding 'Volume' field", zap.Error(err))
			return 0, err
		}

		request = api.GetEC2VolumeCostRequest{
			RegionCode: response.Hits.Hits[0].Source.Location,
			Volume:     volumeStruct,
		}
	} else {
		h.logger.Error("cannot parse resource", zap.String("Description", fmt.Sprintf("%v", response.Hits.Hits[0].Source.Description)))
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
	if description, ok := response.Hits.Hits[0].Source.Description.(map[string]interface{}); ok {
		dbInstanceInterface, dbInstanceExists := description["DBInstance"]
		if !dbInstanceExists {
			h.logger.Error("cannot find 'DBInstance' field in Description", zap.Any("Description", response.Hits.Hits[0].Source.Description))
			return 0, fmt.Errorf("cannot find 'DBInstance' field in Description")
		}

		var dbInstanceStruct aws.RDSDBInstanceDescription
		decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			Result:  &dbInstanceStruct,
			TagName: "json",
		})
		if err != nil {
			h.logger.Error("error creating mapstructure decoder", zap.Error(err))
			return 0, err
		}

		if err := decoder.Decode(dbInstanceInterface); err != nil {
			h.logger.Error("error decoding 'DBInstance' field", zap.Error(err))
			return 0, err
		}

		request = api.GetRDSInstanceRequest{
			RegionCode: response.Hits.Hits[0].Source.Location,
			DBInstance: dbInstanceStruct,
		}
	} else {
		h.logger.Error("cannot parse resource", zap.String("Description", fmt.Sprintf("%v", response.Hits.Hits[0].Source.Description)))
		return 0, fmt.Errorf("cannot parse resource")
	}

	cost, err := h.workspaceClient.GetAWS(&httpclient.Context{UserRole: apiAuth.InternalRole}, "aws_db_instance", request)
	if err != nil {
		h.logger.Error("failed in calculating cost", zap.Error(err))
		return 0, err
	}

	return cost, nil
}
