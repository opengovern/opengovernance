package cost_estimator

import "github.com/kaytu-io/kaytu-engine/pkg/cost-estimator/calculator"

func GetComputeVirtualMachineCost(h *HttpHandler, resourceId string) (cost float64, err error) {
	resource, err := h.GetComputeVirtualMachine(resourceId)
	if err != nil {
		return 0, err
	}

	OSType := resource.Description.VirtualMachine.Properties.StorageProfile.OSDisk.OSType
	location := resource.Description.VirtualMachine.Location
	VMSize := resource.Description.VirtualMachine.Properties.HardwareProfile.VMSize
	cost, err = calculator.AzureCostEstimator(OSType, location, VMSize)
	if err != nil {
		return 0, err
	}
	return cost, nil
}
