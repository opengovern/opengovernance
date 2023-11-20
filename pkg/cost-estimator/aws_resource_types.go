package cost_estimator

import (
	"encoding/json"
	"fmt"
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
	//var description aws.EC2InstanceDescription
	jsonData, err := json.Marshal(response.Hits.Hits[0].Source.Description)
	if err != nil {
		h.logger.Error("failed to marshal request", zap.Error(err))
		return 0, fmt.Errorf("failed to marshal request")
	}
	var mapData map[string]interface{}
	err = json.Unmarshal(jsonData, &mapData)
	if err != nil {
		return 0, err
	}
	request := api.GetEC2InstanceCostRequest{
		RegionCode: response.Hits.Hits[0].Source.Location,
	}
	if launchTemplateData, ok := mapData["LaunchTemplateData"].(map[string]interface{}); ok {
		if ebsOptimized, ok := launchTemplateData["EbsOptimized"].(bool); ok {
			request.EBSOptimized = ebsOptimized
		} else {
			request.EBSOptimized = false
		}
		request.EnabledMonitoring = false
		if enableMonitoring, ok := launchTemplateData["Monitoring"].(map[string]bool); ok {
			if enabled, ok := enableMonitoring["Enabled"]; ok {
				request.EnabledMonitoring = enabled
			}
		}
	}
	req := api.BaseRequest{
		Request:      request,
		ResourceType: "aws_instance",
		ResourceId:   resourceId,
	}
	cost, err := h.workspaceClient.GetAWS(&httpclient.Context{UserRole: apiAuth.InternalRole}, req)
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
	jsonData, err := json.Marshal(response.Hits.Hits[0].Source.Description)
	if err != nil {
		h.logger.Error("failed to marshal request", zap.Error(err))
		return 0, fmt.Errorf("failed to marshal request")
	}
	var mapData map[string]interface{}
	err = json.Unmarshal(jsonData, &mapData)
	if err != nil {
		return 0, err
	}
	var request api.GetEC2VolumeCostRequest
	request.RegionCode = response.Hits.Hits[0].Source.Location
	if volume, ok := mapData["Volume"].(map[string]interface{}); ok {
		if volumeType, ok := volume["VolumeType"].(string); ok {
			request.Type = volumeType
		}
		if size, ok := volume["Size"].(float64); ok {
			request.Size = size
		}
		if iops, ok := volume["Iops"].(float64); ok {
			request.IOPs = iops
		}
	}
	req := api.BaseRequest{
		Request:      request,
		ResourceType: "aws_ebs_volume",
		ResourceId:   resourceId,
	}
	cost, err := h.workspaceClient.GetAWS(&httpclient.Context{UserRole: apiAuth.InternalRole}, req)
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
		jsonData, err := json.Marshal(response.Hits.Hits[0].Source.Description)
		if err != nil {
			h.logger.Error("failed to marshal request", zap.Error(err))
			return 0, fmt.Errorf("failed to marshal request")
		}
		var mapData map[string]interface{}
		err = json.Unmarshal(jsonData, &mapData)
		if err != nil {
			return 0, err
		}
		request = api.GetLBCostRequest{
			RegionCode: response.Hits.Hits[0].Source.Location,
		}
		if loadBalancer, ok := mapData["LoadBalancer"].(map[string]interface{}); ok {
			if lbType, ok := loadBalancer["Type"].(string); ok {
				request.LBType = lbType
			}
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
		request = api.GetLBCostRequest{
			RegionCode: response.Hits.Hits[0].Source.Location,
			LBType:     "classic",
		}
	}
	req := api.BaseRequest{
		Request:      request,
		ResourceType: "aws_elb",
		ResourceId:   resourceId,
	}
	cost, err := h.workspaceClient.GetAWS(&httpclient.Context{UserRole: apiAuth.InternalRole}, req)
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
	jsonData, err := json.Marshal(response.Hits.Hits[0].Source.Description)
	if err != nil {
		h.logger.Error("failed to marshal request", zap.Error(err))
		return 0, fmt.Errorf("failed to marshal request")
	}
	var mapData map[string]interface{}
	err = json.Unmarshal(jsonData, &mapData)
	if err != nil {
		return 0, err
	}
	request := api.GetRDSInstanceRequest{
		RegionCode: response.Hits.Hits[0].Source.Location,
	}
	if dbInstance, ok := mapData["DBInstance"].(map[string]interface{}); ok {
		if engine, ok := dbInstance["Engine"].(string); ok {
			request.InstanceEngine = engine
		}
		if licenseModel, ok := dbInstance["LicenseModel"].(string); ok {
			request.InstanceLicenseModel = licenseModel
		}
		if multiAz, ok := dbInstance["MultiAZ"].(bool); ok {
			request.InstanceMultiAZ = multiAz
		}
		if allocatedStorage, ok := dbInstance["AllocatedStorage"].(float64); ok {
			request.AllocatedStorage = allocatedStorage
		}
		if storageType, ok := dbInstance["StorageType"].(string); ok {
			request.StorageType = storageType
		}
		if iops, ok := dbInstance["Iops"].(float64); ok {
			request.IOPs = iops
		}
	}
	req := api.BaseRequest{
		Request:      request,
		ResourceType: "aws_db_instance",
		ResourceId:   resourceId,
	}
	cost, err := h.workspaceClient.GetAWS(&httpclient.Context{UserRole: apiAuth.InternalRole}, req)
	if err != nil {
		h.logger.Error("failed in calculating cost", zap.Error(err))
		return 0, err
	}

	return cost, nil
}
