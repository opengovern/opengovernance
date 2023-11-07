package cost_estimator

import (
	"github.com/kaytu-io/kaytu-engine/pkg/cost-estimator/calculator/azure"
	"github.com/kaytu-io/kaytu-engine/pkg/cost-estimator/es"
)

func GetComputeVirtualMachineCost(h *HttpHandler, resourceId string) (float64, error) {
	resource, err := es.GetMicrosoftVirtualMachine(h.client, resourceId)
	if err != nil {
		return 0, err
	}

	OSType := resource.Hits.Hits[0].Source.Description.VirtualMachine.Properties.StorageProfile.OSDisk.OSType
	location := resource.Hits.Hits[0].Source.Description.VirtualMachine.Location
	VMSize := resource.Hits.Hits[0].Source.Description.VirtualMachine.Properties.HardwareProfile.VMSize
	cost, err := azure.VirtualMachineCostEstimator(OSType, location, VMSize)
	if err != nil {
		return 0, err
	}
	return cost, nil
}

func GetVirtualNetworkCost(h *HttpHandler, resourceId string) (float64, error) {
	//var resource azureCompute.VirtualNetwork
	//err := h.GetResource("Microsoft.Network/virtualNetworks", resourceId, &resource)
	//if err != nil {
	//	return 0, err
	//}

	return 0, nil
}
