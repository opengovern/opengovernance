package cost_estimator

import (
	"fmt"
	azureModel "github.com/kaytu-io/kaytu-azure-describer/azure/model"
	"github.com/kaytu-io/kaytu-engine/pkg/cost-estimator/calculator/azure"
	"github.com/kaytu-io/kaytu-engine/pkg/cost-estimator/es"
)

func GetComputeVirtualMachineCost(h *HttpHandler, _ string, resourceId string) (float64, error) {

	response, err := es.GetElasticsearch(h.client, resourceId, "Microsoft.Compute/virtualMachines")
	if err != nil {
		return 0, err
	}
	if len(response.Hits.Hits) == 0 {
		return 0, fmt.Errorf("no resource found")
	}
	var vm azureModel.ComputeVirtualMachineDescription
	if resource, ok := response.Hits.Hits[0].Source.Description.(azureModel.ComputeVirtualMachineDescription); ok {
		vm = resource
	} else {
		return 0, fmt.Errorf("cannot parse resource")
	}
	OSType := vm.VirtualMachine.Properties.StorageProfile.OSDisk.OSType
	location := vm.VirtualMachine.Location
	VMSize := vm.VirtualMachine.Properties.HardwareProfile.VMSize
	cost, err := azure.VirtualMachineCostEstimator(OSType, location, VMSize)
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
