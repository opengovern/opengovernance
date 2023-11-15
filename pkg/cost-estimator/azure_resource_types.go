package cost_estimator

import (
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v4"
	azureModel "github.com/kaytu-io/kaytu-azure-describer/azure/model"
	azureSteampipe "github.com/kaytu-io/kaytu-azure-describer/pkg/steampipe"
	apiAuth "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/cost-estimator/es"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/mitchellh/mapstructure"
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
	var request api.GetAzureVmRequest
	if description, ok := response.Hits.Hits[0].Source.Description.(map[string]interface{}); ok {
		vmInterface, vmExists := description["VM"]
		if !vmExists {
			h.logger.Error("cannot find 'VM' field in Description", zap.Any("Description", response.Hits.Hits[0].Source.Description))
			return 0, fmt.Errorf("cannot find 'VM' field in Description")
		}

		var vmStruct azureModel.ComputeVirtualMachineDescription
		decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			Result:  &vmStruct,
			TagName: "json",
		})
		if err != nil {
			h.logger.Error("error creating mapstructure decoder", zap.Error(err))
			return 0, err
		}

		if err := decoder.Decode(vmInterface); err != nil {
			h.logger.Error("error decoding 'VM' field", zap.Error(err))
			return 0, err
		}

		request = api.GetAzureVmRequest{
			RegionCode: response.Hits.Hits[0].Source.Location,
			VM:         vmStruct,
		}
	} else {
		h.logger.Error("cannot parse resource", zap.String("Description", fmt.Sprintf("%v", response.Hits.Hits[0].Source.Description)))
		return 0, fmt.Errorf("cannot parse resource")
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
	var request api.GetAzureManagedStorageRequest
	if description, ok := response.Hits.Hits[0].Source.Description.(map[string]interface{}); ok {
		diskInterface, diskExists := description["Disk"]
		if !diskExists {
			h.logger.Error("cannot find 'Disk' field in Description", zap.Any("Description", response.Hits.Hits[0].Source.Description))
			return 0, fmt.Errorf("cannot find 'Disk' field in Description")
		}

		var diskStruct armcompute.Disk
		decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			Result:  &diskStruct,
			TagName: "json",
		})
		if err != nil {
			h.logger.Error("error creating mapstructure decoder", zap.Error(err))
			return 0, err
		}

		if err := decoder.Decode(diskInterface); err != nil {
			result, err2 := azureSteampipe.AzureDescriptionToRecord(response.Hits.Hits[0].Source.Description.(interface{}), "azure_compute_disk")
			if err2 != nil {
				h.logger.Error("error description to record", zap.Error(err2))
			}
			h.logger.Info("AzureDescriptionToRecord", zap.String("Description", fmt.Sprintf("%v", result)))
			h.logger.Error("error decoding 'Disk' field", zap.Error(err))
			return 0, err
		}

		computeDiskDescription := azureModel.ComputeDiskDescription{
			Disk:          diskStruct,
			ResourceGroup: response.Hits.Hits[0].Source.ResourceGroup,
		}

		request = api.GetAzureManagedStorageRequest{
			RegionCode:     response.Hits.Hits[0].Source.Location,
			ManagedStorage: computeDiskDescription,
		}
	} else {
		h.logger.Error("cannot parse resource", zap.String("Description", fmt.Sprintf("%v", response.Hits.Hits[0].Source.Description)))
		return 0, fmt.Errorf("cannot parse resource")
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
	var request api.GetAzureLoadBalancerRequest
	if description, ok := response.Hits.Hits[0].Source.Description.(map[string]interface{}); ok {
		lbInterface, lbExists := description["LoadBalancer"]
		if !lbExists {
			h.logger.Error("cannot find 'LoadBalancer' field in Description", zap.Any("Description", response.Hits.Hits[0].Source.Description))
			return 0, fmt.Errorf("cannot find 'LoadBalancer' field in Description")
		}

		var lbStruct azureModel.LoadBalancerDescription
		decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			Result:  &lbStruct,
			TagName: "json",
		})
		if err != nil {
			h.logger.Error("error creating mapstructure decoder", zap.Error(err))
			return 0, err
		}

		if err := decoder.Decode(lbInterface); err != nil {
			h.logger.Error("error decoding 'LoadBalancer' field", zap.Error(err))
			return 0, err
		}

		request = api.GetAzureLoadBalancerRequest{
			RegionCode:   response.Hits.Hits[0].Source.Location,
			LoadBalancer: lbStruct,
		}
	} else {
		h.logger.Error("cannot parse resource", zap.String("Description", fmt.Sprintf("%v", response.Hits.Hits[0].Source.Description)))
		return 0, fmt.Errorf("cannot parse resource")
	}

	cost, err := h.workspaceClient.GetAzure(&httpclient.Context{UserRole: apiAuth.InternalRole}, "", request)
	if err != nil {
		h.logger.Error("failed in calculating cost", zap.Error(err))
		return 0, err
	}

	return cost, nil
}

func GetVirtualNetworkCost(h *HttpHandler, _ string, resourceId string) (float64, error) {
	//var resource azureCompute.VirtualNetwork
	//err := h.GetResource("Microsoft.Network/virtualNetworks", resourceId, &resource)
	//if err != nil {
	//	return 0, err
	//}

	return 0, nil
}
