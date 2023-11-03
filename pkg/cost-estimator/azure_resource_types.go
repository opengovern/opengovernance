package cost_estimator

import (
	azureCompute "github.com/kaytu-io/kaytu-azure-describer/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-engine/pkg/cost-estimator/calculator/azure"
)

func GetComputeVirtualMachineCost(h *HttpHandler, resourceId string, timeInterval int64) (float64, error) {
	var resource azureCompute.ComputeVirtualMachine
	//err := h.GetResource("Microsoft.Compute/virtualMachines", resourceId, &resource)
	//if err != nil {
	//	return 0, err
	//}

	OSType := resource.Description.VirtualMachine.Properties.StorageProfile.OSDisk.OSType
	location := resource.Description.VirtualMachine.Location
	VMSize := resource.Description.VirtualMachine.Properties.HardwareProfile.VMSize
	cost, err := azure.VirtualMachineCostEstimator(OSType, location, VMSize)
	if err != nil {
		return 0, err
	}
	return cost, nil
}

func GetVirtualNetworkCost(h *HttpHandler, resourceId string, timeInterval int64) (float64, error) {
	//var resource azureCompute.VirtualNetwork
	//err := h.GetResource("Microsoft.Network/virtualNetworks", resourceId, &resource)
	//if err != nil {
	//	return 0, err
	//}

	return 0, nil
}
