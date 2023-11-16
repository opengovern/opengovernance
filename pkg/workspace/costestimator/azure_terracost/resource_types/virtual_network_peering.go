package resource_types

import "github.com/kaytu-io/kaytu-engine/pkg/workspace/api"

// VirtualNetworkPeering is the entity that holds the logic to calculate price
// of the azure_virtualnetwork_peering
type VirtualNetworkPeering struct {
	provider *Provider

	location        string
	vmSize          string
	operatingSystem OS
}

// virtualNetworkPeeringValues is holds the values that we need to be able
// to calculate the price of the Virtual Network Values
type virtualNetworkPeeringValues struct {
	VMSize          string `mapstructure:"vm_size"`
	Location        string `mapstructure:"location"`
	OperatingSystem OS     `mapstructure:"operating_system"`
}

// decodeVirtualMachineValues decodes and returns computeInstanceValues from a Terraform values map.
func decodeVirtualNetworkPeeringValues(request api.GetAzureVmRequest) virtualMachineValues {
	return virtualMachineValues{
		VMSize:          string(*request.VM.VirtualMachine.Properties.HardwareProfile.VMSize),
		Location:        request.RegionCode,
		OperatingSystem: OS(*request.VM.VirtualMachine.Properties.StorageProfile.OSDisk.OSType),
	}
}
