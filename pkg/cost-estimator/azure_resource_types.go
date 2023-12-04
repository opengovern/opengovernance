package cost_estimator

import (
	"encoding/json"
	"fmt"
	azure "github.com/kaytu-io/kaytu-azure-describer/azure/model"
	apiAuth "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/cost-estimator/es"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"go.uber.org/zap"
)

func GetComputeVirtualMachineCost(h *HttpHandler, _ string, resourceId string) (float64, error) {
	response, err := es.GetElasticsearch(h.logger, h.client, resourceId, "Microsoft.Compute/virtualMachines")
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
	var request api.GetAzureVmRequest
	h.logger.Info("MapData", zap.String("MapData", fmt.Sprintf("%v", mapData)))

	request.RegionCode = response.Hits.Hits[0].Source.Location
	if virtualMachine, ok := mapData["VirtualMachine"].(map[string]interface{}); ok {
		if properties, ok := virtualMachine["Properties"].(map[string]interface{}); ok {
			if storageProfile, ok := properties["StorageProfile"].(map[string]interface{}); ok {
				if osDisk, ok := storageProfile["OSDisk"].(map[string]interface{}); ok {
					if osType, ok := osDisk["OSType"].(string); ok {
						request.OperatingSystem = osType
					}
				}
			}
			if hardwareProfile, ok := properties["HardwareProfile"].(map[string]interface{}); ok {
				if vmSize, ok := hardwareProfile["VMSize"].(string); ok {
					request.VMSize = vmSize
				}
			}
		}
	}
	req := api.BaseRequest{
		Request:      request,
		ResourceType: "azurerm_virtual_machine",
		ResourceId:   resourceId,
	}
	h.logger.Info("request", zap.String("Request", fmt.Sprintf("%v", req)))
	cost, err := h.workspaceClient.GetAzure(&httpclient.Context{UserRole: apiAuth.InternalRole}, req)
	if err != nil {
		h.logger.Error("failed in calculating cost", zap.Error(err))
		return 0, err
	}

	return cost, nil
}

func GetManagedStorageCost(h *HttpHandler, _ string, resourceId string) (float64, error) {
	response, err := es.GetElasticsearch(h.logger, h.client, resourceId, "Microsoft.Compute/disks")
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
	var request api.GetAzureManagedStorageRequest
	request.RegionCode = response.Hits.Hits[0].Source.Location
	h.logger.Info("MapData", zap.String("MapData", fmt.Sprintf("%v", mapData)))
	if disk, ok := mapData["Disk"].(map[string]interface{}); ok {
		if skuName, ok := disk["SKU"].(map[string]interface{})["Name"].(string); ok {
			request.SkuName = skuName
		}
		if properties, ok := disk["Properties"].(map[string]interface{}); ok {
			request.DiskSize = properties["DiskSizeGB"].(float64)
			if fmt.Sprintf("%v", properties["BurstingEnabled"]) == "true" {
				request.BurstingEnabled = true
			} else {
				request.BurstingEnabled = false
			}
			request.DiskThroughput = properties["DiskMBpsReadWrite"].(float64)
			request.DiskIOPs = properties["DiskIOPSReadWrite"].(float64)
		}
	}
	req := api.BaseRequest{
		Request:      request,
		ResourceId:   resourceId,
		ResourceType: "azurerm_managed_disk",
	}
	h.logger.Info("request", zap.String("Request", fmt.Sprintf("%v", req)))
	cost, err := h.workspaceClient.GetAzure(&httpclient.Context{UserRole: apiAuth.InternalRole}, req)
	if err != nil {
		h.logger.Error("failed in calculating cost", zap.Error(err))
		return 0, err
	}

	return cost, nil
}

func GetLoadBalancerCost(h *HttpHandler, _ string, resourceId string) (float64, error) {
	response, err := es.GetElasticsearch(h.logger, h.client, resourceId, "Microsoft.Network/loadBalancers")
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
	var request api.GetAzureLoadBalancerRequest
	request.RegionCode = response.Hits.Hits[0].Source.Location
	h.logger.Info("MapData", zap.String("MapData", fmt.Sprintf("%v", mapData)))

	if loadBalancer, ok := mapData["LoadBalancer"].(map[string]interface{}); ok {
		if properties, ok := loadBalancer["Properties"].(map[string]interface{}); ok {
			var rulesNumber int
			if loadBalancingRules, ok := properties["LoadBalancingRules"].([]map[string]interface{}); ok {
				rulesNumber = rulesNumber + len(loadBalancingRules)
			}
			if outboundRules, ok := properties["OutboundRules"].([]map[string]interface{}); ok {
				rulesNumber = rulesNumber + len(outboundRules)
			}
			request.RulesNumber = int32(rulesNumber)
		}
		if sku, ok := loadBalancer["SKU"].(map[string]string); ok {
			if name, ok := sku["Name"]; ok {
				request.SkuName = name
			}
			if tier, ok := sku["Tier"]; ok {
				request.SkuTier = tier
			}
		}
	}
	req := api.BaseRequest{
		Request:      request,
		ResourceType: "azurerm_load_balancer",
		ResourceId:   resourceId,
	}
	h.logger.Info("request", zap.String("Request", fmt.Sprintf("%v", req)))
	cost, err := h.workspaceClient.GetAzure(&httpclient.Context{UserRole: apiAuth.InternalRole}, req)
	if err != nil {
		h.logger.Error("failed in calculating cost", zap.Error(err))
		return 0, err
	}

	return cost, nil
}

func GetVirtualNetworkCost(h *HttpHandler, _ string, resourceId string) (float64, error) {
	response, err := es.GetElasticsearch(h.logger, h.client, resourceId, "Microsoft.Network/virtualNetworks")
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
	var peeringLocations []string
	h.logger.Info("MapData", zap.String("MapData", fmt.Sprintf("%v", mapData)))

	if virtualNetwork, ok := mapData["VirtualNetwork"].(map[string]interface{}); ok {
		if properties, ok := virtualNetwork["Properties"].(map[string]interface{}); ok {
			if peerings, ok := properties["VirtualNetworkPeerings"].([]map[string]interface{}); ok {
				for _, p := range peerings {
					if peeringProperties, ok := p["Properties"].(map[string]interface{}); ok {
						if remoteVirtualNetwork, ok := peeringProperties["RemoteVirtualNetwork"].(map[string]string); ok {
							if id, ok := remoteVirtualNetwork["ID"]; ok {
								location, err := getVirtualNetworkPeering(h, id)
								if err != nil {
									return 0, err
								}
								peeringLocations = append(peeringLocations, *location)
							}
						}
					}
				}
			}
		}
	}
	request := api.GetAzureVirtualNetworkRequest{
		RegionCode:       response.Hits.Hits[0].Source.Location,
		PeeringLocations: peeringLocations,
	}
	req := api.BaseRequest{
		Request:      request,
		ResourceType: "azurerm_virtual_network",
		ResourceId:   resourceId,
	}
	h.logger.Info("request", zap.String("Request", fmt.Sprintf("%v", req)))
	cost, err := h.workspaceClient.GetAzure(&httpclient.Context{UserRole: apiAuth.InternalRole}, req)
	if err != nil {
		h.logger.Error("failed in calculating cost", zap.Error(err))
		return 0, err
	}
	return cost, nil
}

func getVirtualNetworkPeering(h *HttpHandler, resourceId string) (*string, error) {
	response, err := es.GetElasticsearch(h.logger, h.client, resourceId, "Microsoft.Network/virtualNetworks")
	if err != nil {
		h.logger.Error("failed to get resource", zap.Error(err))
		return nil, fmt.Errorf("failed to get resource")
	}
	if len(response.Hits.Hits) == 0 {
		return nil, fmt.Errorf("no resource found")
	}
	return &response.Hits.Hits[0].Source.Location, nil
}

func GetSQLDatabaseCost(h *HttpHandler, _ string, resourceId string) (float64, error) {
	response, err := es.GetElasticsearch(h.logger, h.client, resourceId, "Microsoft.Sql/servers/databases")
	if err != nil {
		return 0, err
	}
	if len(response.Hits.Hits) == 0 {
		return 0, fmt.Errorf("no resource found")
	}
	var request api.GetAzureSqlServersDatabasesRequest
	if sqlServerDB, ok := response.Hits.Hits[0].Source.Description.(azure.SqlDatabaseDescription); ok {
		request = api.GetAzureSqlServersDatabasesRequest{
			RegionCode:  response.Hits.Hits[0].Source.Location,
			SqlServerDB: sqlServerDB,
		}
	} else {
		return 0, fmt.Errorf("cannot parse resource")
	}
	req := api.BaseRequest{
		Request:      request,
		ResourceType: "azurerm_sql_server_DB",
		ResourceId:   resourceId,
	}
	cost, err := h.workspaceClient.GetAzure(&httpclient.Context{UserRole: apiAuth.InternalRole}, req)
	if err != nil {
		return 0, err
	}
	return cost, nil
}
