package cost_estimator

import (
	"encoding/json"
	"fmt"
	azure "github.com/kaytu-io/kaytu-azure-describer/azure/model"
	apiAuth "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/cost-estimator/es"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"go.uber.org/zap"
	"strings"
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
	var description azure.ComputeVirtualMachineDescription
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
	request := api.GetAzureVmRequest{
		RegionCode: response.Hits.Hits[0].Source.Location,
		VM:         description,
	}
	cost, err := h.workspaceClient.GetAzure(&httpclient.Context{UserRole: apiAuth.InternalRole}, "azurerm_virtual_machine", request)
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
	var description azure.ComputeDiskDescription
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
	h.logger.Info("Compute Disk", zap.String("elasticsearch", fmt.Sprintf("%v", response.Hits.Hits[0].Source.Description)))
	h.logger.Info("Compute Disk", zap.String("jsonData", string(jsonData)))
	jsonData = []byte(strings.ReplaceAll(string(jsonData), "\\\"", "\""))
	h.logger.Info("Compute Disk", zap.String("jsonData CONVERTED", string(jsonData)))
	h.logger.Info("Compute Disk", zap.String("description", fmt.Sprintf("%v", description)))
	request := api.GetAzureManagedStorageRequest{
		RegionCode:     response.Hits.Hits[0].Source.Location,
		ManagedStorage: description,
	}
	cost, err := h.workspaceClient.GetAzure(&httpclient.Context{UserRole: apiAuth.InternalRole}, "azurerm_managed_disk", request)
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
	var description azure.LoadBalancerDescription
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
	request := api.GetAzureLoadBalancerRequest{
		RegionCode:   response.Hits.Hits[0].Source.Location,
		LoadBalancer: description,
	}
	cost, err := h.workspaceClient.GetAzure(&httpclient.Context{UserRole: apiAuth.InternalRole}, "azurerm_load_balancer", request)
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
	var description azure.VirtualNetworkDescription
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
	var peeringLocations []string
	for _, p := range description.VirtualNetwork.Properties.VirtualNetworkPeerings {
		id := *p.Properties.RemoteVirtualNetwork.ID
		location, err := getVirtualNetworkPeering(h, id)
		if err != nil {
			h.logger.Error(fmt.Sprintf("can not get virtual network peering %s", id))
			return 0, fmt.Errorf("can not get virtual network peering %s", id)
		}
		peeringLocations = append(peeringLocations, *location)
	}
	request := api.GetAzureVirtualNetworkRequest{
		RegionCode:       response.Hits.Hits[0].Source.Location,
		PeeringLocations: peeringLocations,
	}
	cost, err := h.workspaceClient.GetAzure(&httpclient.Context{UserRole: apiAuth.InternalRole}, "azurerm_virtual_network", request)
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
	var description azure.VirtualNetworkDescription
	jsonData, err := json.Marshal(response.Hits.Hits[0].Source.Description)
	if err != nil {
		h.logger.Error("failed to marshal request", zap.Error(err))
		return nil, fmt.Errorf("failed to marshal request")
	}
	err = json.Unmarshal(jsonData, &description)
	if err != nil {
		h.logger.Error("cannot parse resource", zap.String("interface",
			fmt.Sprintf("%v", string(jsonData))))
		return nil, fmt.Errorf("cannot parse resource %s", err.Error())
	}
	return description.VirtualNetwork.Location, nil
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
			ResourceId:  resourceId,
		}
	} else {
		return 0, fmt.Errorf("cannot parse resource")
	}

	cost, err := h.workspaceClient.GetAzureSqlServerDatabase(&httpclient.Context{UserRole: apiAuth.InternalRole}, request)
	if err != nil {
		return 0, err
	}
	return cost, nil
}
