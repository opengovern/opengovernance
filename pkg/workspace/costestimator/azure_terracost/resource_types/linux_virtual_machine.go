package resource_types

import (
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/price"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/product"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/query"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/util"
	"github.com/shopspring/decimal"
)

// LinuxVirtualMachine is the entity that holds the logic to calculate price
// of the google_compute_instance
type LinuxVirtualMachine struct {
	provider *Provider

	location string
	size     string
}

// linuxVirtualMachineValues is holds the values that we need to be able
// to calculate the price of the ComputeInstance
type linuxVirtualMachineValues struct {
	Size     string `mapstructure:"size"`
	Location string `mapstructure:"location"`
}

// decodeLinuxVirtualMachineValues decodes and returns computeInstanceValues from a Terraform values map.
func decodeLinuxVirtualMachineValues(request api.GetAzureVmRequest) linuxVirtualMachineValues {
	return linuxVirtualMachineValues{
		Size:     string(*request.VM.VirtualMachine.Properties.HardwareProfile.VMSize),
		Location: request.RegionCode,
	}
}

// newLinuxVirtualMachine initializes a new LinuxVirtualMachine from the provider
func (p *Provider) newLinuxVirtualMachine(vals linuxVirtualMachineValues) *LinuxVirtualMachine {
	inst := &LinuxVirtualMachine{
		provider: p,

		location: getLocationName(vals.Location),
		size:     vals.Size,
	}

	return inst
}

// Components returns the price component queries that make up this Instance.
func (inst *LinuxVirtualMachine) Components() []query.Component {
	components := []query.Component{inst.linuxVirtualMachineComponent()}

	return components
}

// linuxVirtualMachineComponent returns the query needed to be able to calculate the price
func (inst *LinuxVirtualMachine) linuxVirtualMachineComponent() query.Component {
	return linuxVirtualMachineComponent(inst.provider.key, inst.location, inst.size)
}

// linuxVirtualMachineComponent is the abstraction of the same LinuxVirtualMachine.linuxVirtualMachineComponent
// so it can be reused
func linuxVirtualMachineComponent(key, location, size string) query.Component {
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
			},
		},
		PriceFilter: &price.Filter{
			Unit: util.StringPtr("1 Hour"),
		},
	}
}
