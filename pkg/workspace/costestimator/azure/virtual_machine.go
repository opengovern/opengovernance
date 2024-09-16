package azure

import (
	"github.com/kaytu-io/open-governance/pkg/workspace/api"
	"github.com/kaytu-io/open-governance/pkg/workspace/costestimator/price"
	"github.com/kaytu-io/open-governance/pkg/workspace/costestimator/product"
	"github.com/kaytu-io/open-governance/pkg/workspace/costestimator/query"
	"github.com/kaytu-io/open-governance/pkg/workspace/costestimator/util"
	"github.com/shopspring/decimal"
)

type OS string

const (
	WindowsOS OS = "Windows"
	LinuxOS   OS = "Linux"
)

// VirtualMachine is the entity that holds the logic to calculate price
// of the google_compute_instance
type VirtualMachine struct {
	provider *Provider

	location        string
	vmSize          string
	operatingSystem OS
}

// virtualMachineValues is holds the values that we need to be able
// to calculate the price of the ComputeInstance
type virtualMachineValues struct {
	VMSize          string `mapstructure:"vm_size"`
	Location        string `mapstructure:"location"`
	OperatingSystem OS     `mapstructure:"operating_system"`
}

// decodeVirtualMachineValues decodes and returns computeInstanceValues
func decodeVirtualMachineValues(request api.GetAzureVmRequest) virtualMachineValues {
	return virtualMachineValues{
		VMSize:          request.VMSize,
		Location:        request.RegionCode,
		OperatingSystem: OS(request.OperatingSystem),
	}
}

// newVirtualMachine initializes a new VirtualMachine from the provider
func (p *Provider) newVirtualMachine(vals virtualMachineValues) *VirtualMachine {
	inst := &VirtualMachine{
		provider: p,

		location:        getLocationName(vals.Location),
		vmSize:          vals.VMSize,
		operatingSystem: vals.OperatingSystem,
	}

	return inst
}

// Components returns the price component queries that make up this Instance.
func (inst *VirtualMachine) Components() []query.Component {
	var components []query.Component
	if inst.operatingSystem == WindowsOS {
		components = []query.Component{inst.virtualMachineComponent(inst.provider.key, inst.location, inst.vmSize, ".*Windows.*")}
	} else if inst.operatingSystem == LinuxOS {
		components = []query.Component{inst.virtualMachineComponent(inst.provider.key, inst.location, inst.vmSize, "^(?!.*Windows).*")}
	}

	return components
}

// linuxVirtualMachineComponent is the abstraction of the same LinuxVirtualMachine.linuxVirtualMachineComponent
// so it can be reused
func (inst *VirtualMachine) virtualMachineComponent(key, location, size string, osRegex string) query.Component {
	return query.Component{
		Name:           "Compute",
		HourlyQuantity: decimal.NewFromInt(1),
		ProductFilter: &product.Filter{
			Provider: util.StringPtr(key),
			Service:  util.StringPtr("Virtual Machines"),
			Family:   util.StringPtr("Compute"),
			Location: util.StringPtr(location),
			AttributeFilters: []*product.AttributeFilter{
				{Key: "arm_sku_name", Value: util.StringPtr(size)},
				{Key: "priority", Value: util.StringPtr("regular")},
				{Key: "product_name", ValueRegex: util.StringPtr(osRegex)},
			},
		},
		PriceFilter: &price.Filter{
			Unit: util.StringPtr("1 Hour"),
		},
	}
}
