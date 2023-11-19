package cost_estimator

import (
	"encoding/json"
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
	var description aws.EC2InstanceDescription
	jsonData, err := json.Marshal(response.Hits.Hits[0].Source.Description)
	if err != nil {
		h.logger.Error("failed to marshal request", zap.Error(err))
		return 0, fmt.Errorf("failed to marshal request")
	}
	err = json.Unmarshal(jsonData, &description)
	if err != nil {
		h.logger.Error("cannot parse resource", zap.String("interface",
			fmt.Sprintf("%v", string(jsonData))))
		return 0, fmt.Errorf("cannot parse resource %s", err.Error())
	}
	request := api.GetEC2InstanceCostRequest{
		RegionCode: response.Hits.Hits[0].Source.Location,
		Instance:   description,
	}

	cost, err := h.workspaceClient.GetAWS(&httpclient.Context{UserRole: apiAuth.InternalRole}, "aws_instance", struct {
		request    any
		resourceId string
	}{
		request:    request,
		resourceId: resourceId,
	})
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
	var description aws.EC2VolumeDescription
	jsonData, err := json.Marshal(response.Hits.Hits[0].Source.Description)
	if err != nil {
		h.logger.Error("failed to marshal request", zap.Error(err))
		return 0, fmt.Errorf("failed to marshal request")
	}
	err = json.Unmarshal(jsonData, &description)
	if err != nil {
		h.logger.Error("cannot parse resource", zap.String("interface",
			fmt.Sprintf("%v", string(jsonData))))
		return 0, fmt.Errorf("cannot parse resource %s", err.Error())
	}
	request := api.GetEC2VolumeCostRequest{
		RegionCode: response.Hits.Hits[0].Source.Location,
		Volume:     description,
	}

	cost, err := h.workspaceClient.GetAWS(&httpclient.Context{UserRole: apiAuth.InternalRole}, "aws_ebs_volume", struct {
		request    any
		resourceId string
	}{
		request:    request,
		resourceId: resourceId,
	})
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
		var description aws.ElasticLoadBalancingV2LoadBalancerDescription
		jsonData, err := json.Marshal(response.Hits.Hits[0].Source.Description)
		if err != nil {
			h.logger.Error("failed to marshal request", zap.Error(err))
			return 0, fmt.Errorf("failed to marshal request")
		}
		err = json.Unmarshal(jsonData, &description)
		if err != nil {
			h.logger.Error("cannot parse resource", zap.String("interface",
				fmt.Sprintf("%v", string(jsonData))))
			return 0, fmt.Errorf("cannot parse resource %s", err.Error())
		}
		request = api.GetLBCostRequest{
			RegionCode: response.Hits.Hits[0].Source.Location,
			LBType:     string(description.LoadBalancer.Type),
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
		var description aws.ElasticLoadBalancingLoadBalancerDescription
		jsonData, err := json.Marshal(response.Hits.Hits[0].Source.Description)
		if err != nil {
			h.logger.Error("failed to marshal request", zap.Error(err))
			return 0, fmt.Errorf("failed to marshal request")
		}
		err = json.Unmarshal(jsonData, &description)
		if err != nil {
			h.logger.Error("cannot parse resource", zap.String("interface",
				fmt.Sprintf("%v", string(jsonData))))
			return 0, fmt.Errorf("cannot parse resource %s", err.Error())
		}
		request = api.GetLBCostRequest{
			RegionCode: response.Hits.Hits[0].Source.Location,
			LBType:     "classic",
		}
	}
	cost, err := h.workspaceClient.GetAWS(&httpclient.Context{UserRole: apiAuth.InternalRole}, "aws_elb", struct {
		request    any
		resourceId string
	}{
		request:    request,
		resourceId: resourceId,
	})
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
	var description aws.RDSDBInstanceDescription
	jsonData, err := json.Marshal(response.Hits.Hits[0].Source.Description)
	if err != nil {
		h.logger.Error("failed to marshal request", zap.Error(err))
		return 0, fmt.Errorf("failed to marshal request")
	}
	err = json.Unmarshal(jsonData, &description)
	if err != nil {
		h.logger.Error("cannot parse resource", zap.String("interface",
			fmt.Sprintf("%v", string(jsonData))))
		return 0, fmt.Errorf("cannot parse resource %s", err.Error())
	}
	request := api.GetRDSInstanceRequest{
		RegionCode: response.Hits.Hits[0].Source.Location,
		DBInstance: description,
	}

	cost, err := h.workspaceClient.GetAWS(&httpclient.Context{UserRole: apiAuth.InternalRole}, "aws_db_instance", struct {
		request    any
		resourceId string
	}{
		request:    request,
		resourceId: resourceId,
	})
	if err != nil {
		h.logger.Error("failed in calculating cost", zap.Error(err))
		return 0, err
	}

	return cost, nil
}
