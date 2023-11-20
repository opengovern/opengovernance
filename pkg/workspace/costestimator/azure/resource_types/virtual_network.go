package resource_types

import (
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/query"
)

// VirtualNetwork is the entity that holds the logic to calculate price
// of the azure_network_virtualnetwork
type VirtualNetwork struct {
	provider *Provider

	location              string
	peeringLocations      []string
	monthlyDataTransferGB float64
}

// virtualNetworkValues is holds the values that we need to be able
// to calculate the price of the Virtual Network Values
type virtualNetworkValues struct {
	Location              string   `mapstructure:"location"`
	PeeringLocations      []string `mapstructure:"peering_locations"`
	MonthlyDataTransferGB float64  `mapstructure:"monthly_data_transfer_gb"`
}

// decodeVirtualNetworkValues decodes and returns virtual network values
func decodeVirtualNetworkValues(request api.GetAzureVirtualNetworkRequest) virtualNetworkValues {
	vnValues := virtualNetworkValues{
		Location:              request.RegionCode,
		PeeringLocations:      request.PeeringLocations,
		MonthlyDataTransferGB: 100,
	}
	if request.MonthlyDataTransferGB != nil {
		vnValues.MonthlyDataTransferGB = *request.MonthlyDataTransferGB
	}
	return vnValues
}

func (p *Provider) newVirtualNetwork(vals virtualNetworkValues) *VirtualNetwork {
	inst := &VirtualNetwork{
		provider: p,

		location:              vals.Location,
		peeringLocations:      vals.PeeringLocations,
		monthlyDataTransferGB: vals.MonthlyDataTransferGB,
	}
	return inst
}

func (inst *VirtualNetwork) Components() []query.Component {
	var components []query.Component

	for _, loc := range inst.peeringLocations {
		vals := decodeVirtualNetworkPeeringValues(api.GetAzureVirtualNetworkPeeringRequest{
			SourceLocation:        inst.location,
			DestinationLocation:   loc,
			MonthlyDataTransferGB: &inst.monthlyDataTransferGB,
		})
		peering := inst.provider.newVirtualNetworkPeering(vals)
		components = append(components, peering.Components()...)
	}

	return components
}
