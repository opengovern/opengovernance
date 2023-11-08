package cost_estimator

import (
	"fmt"
	azureModel "github.com/kaytu-io/kaytu-azure-describer/azure/model"
	apiAuth "github.com/kaytu-io/kaytu-engine/pkg/auth/api"
	"github.com/kaytu-io/kaytu-engine/pkg/cost-estimator/es"
	"github.com/kaytu-io/kaytu-engine/pkg/internal/httpclient"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
)

func GetComputeVirtualMachineCost(h *HttpHandler, _ string, resourceId string) (float64, error) {
	response, err := es.GetElasticsearch(h.client, resourceId, "Microsoft.Compute/virtualMachines")
	if err != nil {
		return 0, err
	}
	if len(response.Hits.Hits) == 0 {
		return 0, fmt.Errorf("no resource found")
	}
	var request api.GetAzureVmRequest
	if vm, ok := response.Hits.Hits[0].Source.Description.(azureModel.ComputeVirtualMachineDescription); ok {
		request = api.GetAzureVmRequest{
			RegionCode: response.Hits.Hits[0].Source.Location,
			VM:         vm,
		}
	} else {
		return 0, fmt.Errorf("cannot parse resource")
	}
	cost, err := h.workspaceClient.GetAzureVm(&httpclient.Context{UserRole: apiAuth.InternalRole}, request)
	if err != nil {
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
