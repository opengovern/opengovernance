package azure

import (
	"github.com/opengovern/opengovernance/pkg/workspace/api"
	"github.com/opengovern/opengovernance/pkg/workspace/costestimator/product"
	"github.com/opengovern/opengovernance/pkg/workspace/costestimator/query"
	"github.com/opengovern/opengovernance/pkg/workspace/costestimator/util"
	"github.com/shopspring/decimal"
	"strings"
)

// VirtualNetworkPeering is the entity that holds the logic to calculate price
// of the azure_network_virtualnetwork_peering
type VirtualNetworkPeering struct {
	provider *Provider

	sourceLocation        string
	destinationLocation   string
	monthlyDataTransferGB float64
}

// virtualNetworkPeeringValues is holds the values that we need to be able
// to calculate the price of the Virtual Network Peering Values
type virtualNetworkPeeringValues struct {
	SourceLocation        string  `mapstructure:"source_location"`
	DestinationLocation   string  `mapstructure:"destination_location"`
	MonthlyDataTransferGB float64 `mapstructure:"monthly_data_transfer_gb"`
}

// decodeVirtualNetworkPeeringValues decodes and returns virtual network values
func decodeVirtualNetworkPeeringValues(request api.GetAzureVirtualNetworkPeeringRequest) virtualNetworkPeeringValues {
	vnValues := virtualNetworkPeeringValues{
		SourceLocation:        request.SourceLocation,
		DestinationLocation:   request.DestinationLocation,
		MonthlyDataTransferGB: 100,
	}
	if request.MonthlyDataTransferGB != nil {
		vnValues.MonthlyDataTransferGB = *request.MonthlyDataTransferGB
	}
	return vnValues
}

func (p *Provider) newVirtualNetworkPeering(vals virtualNetworkPeeringValues) *VirtualNetworkPeering {
	inst := &VirtualNetworkPeering{
		provider: p,

		sourceLocation:        vals.SourceLocation,
		destinationLocation:   vals.DestinationLocation,
		monthlyDataTransferGB: vals.MonthlyDataTransferGB,
	}
	return inst
}

func (inst *VirtualNetworkPeering) Components() []query.Component {
	components := []query.Component{
		*inst.egressDataProcessedCostComponent(inst.provider.key),
		*inst.ingressDataProcessedCostComponent(inst.provider.key),
	}

	return components
}

func (inst *VirtualNetworkPeering) egressDataProcessedCostComponent(key string) *query.Component {
	if inst.sourceLocation == inst.destinationLocation {
		return &query.Component{
			Name:            "Outbound data transfer",
			Unit:            "GB",
			MonthlyQuantity: decimal.NewFromInt(int64(inst.monthlyDataTransferGB)),
			ProductFilter: &product.Filter{
				Provider: util.StringPtr(key),
				Service:  util.StringPtr("Virtual Network"),
				Family:   util.StringPtr("Networking"),
				Location: util.StringPtr("Global"),
				AttributeFilters: []*product.AttributeFilter{
					{Key: "meter_name", Value: util.StringPtr("Intra-Region Egress")},
				},
			},
		}
	}

	return &query.Component{
		Name:            "Outbound data transfer",
		Unit:            "GB",
		MonthlyQuantity: decimal.NewFromInt(int64(inst.monthlyDataTransferGB)),
		ProductFilter: &product.Filter{
			Provider: util.StringPtr(key),
			Location: util.StringPtr(virtualNetworkPeeringConvertRegion(inst.sourceLocation)),
			Service:  util.StringPtr("VPN Gateway"),
			Family:   util.StringPtr("Networking"),
			AttributeFilters: []*product.AttributeFilter{
				{Key: "product_name", ValueRegex: util.StringPtr("VPN Gateway Bandwidth")},
				{Key: "meter_name", ValueRegex: util.StringPtr("Inter-Virtual Network Data Transfer Out")},
			},
		},
	}
}

func (inst *VirtualNetworkPeering) ingressDataProcessedCostComponent(key string) *query.Component {
	if inst.sourceLocation == inst.destinationLocation {
		return &query.Component{
			Name:            "Inbound data transfer",
			Unit:            "GB",
			MonthlyQuantity: decimal.NewFromInt(int64(inst.monthlyDataTransferGB)),
			ProductFilter: &product.Filter{
				Provider: util.StringPtr(key),
				Location: util.StringPtr("Global"),
				Service:  util.StringPtr("Virtual Network"),
				Family:   util.StringPtr("Networking"),
				AttributeFilters: []*product.AttributeFilter{
					{Key: "meter_name", Value: util.StringPtr("Intra-Region Ingress")},
				},
			},
		}
	}

	return &query.Component{
		Name:            "Inbound data transfer",
		Unit:            "GB",
		MonthlyQuantity: decimal.NewFromInt(int64(inst.monthlyDataTransferGB)),
		ProductFilter: &product.Filter{
			Provider: util.StringPtr(key),
			Location: util.StringPtr(virtualNetworkPeeringConvertRegion(inst.destinationLocation)),
			Service:  util.StringPtr("VPN Gateway"),
			Family:   util.StringPtr("Networking"),
			AttributeFilters: []*product.AttributeFilter{
				{Key: "product_name", ValueRegex: util.StringPtr("VPN Gateway Bandwidth")},
				{Key: "meter_name", ValueRegex: util.StringPtr("Inter-Virtual Network Data Transfer Out")},
			},
		},
	}
}

func virtualNetworkPeeringConvertRegion(region string) string {
	zone := regionToZone(region)

	if strings.HasPrefix(strings.ToLower(region), "usgov") {
		zone = "US Gov Zone 1"
	}
	if strings.HasPrefix(strings.ToLower(region), "germany") {
		zone = "DE Zone 1"
	}
	if strings.HasPrefix(strings.ToLower(region), "china") {
		zone = "CN Zone 1"
	}

	return zone
}

func regionToZone(region string) string {
	return map[string]string{
		"westus":             "Zone 1",
		"westus2":            "Zone 1",
		"eastus":             "Zone 1",
		"centralus":          "Zone 1",
		"centraluseuap":      "Zone 1",
		"southcentralus":     "Zone 1",
		"northcentralus":     "Zone 1",
		"westcentralus":      "Zone 1",
		"eastus2":            "Zone 1",
		"eastus2euap":        "Zone 1",
		"brazilsouth":        "Zone 3",
		"brazilus":           "Zone 3",
		"northeurope":        "Zone 1",
		"westeurope":         "Zone 1",
		"eastasia":           "Zone 2",
		"southeastasia":      "Zone 2",
		"japanwest":          "Zone 2",
		"japaneast":          "Zone 2",
		"koreacentral":       "Zone 2",
		"koreasouth":         "Zone 2",
		"southindia":         "Zone 5",
		"westindia":          "Zone 5",
		"centralindia":       "Zone 5",
		"australiaeast":      "Zone 4",
		"australiasoutheast": "Zone 4",
		"canadacentral":      "Zone 1",
		"canadaeast":         "Zone 1",
		"uksouth":            "Zone 1",
		"ukwest":             "Zone 1",
		"francecentral":      "Zone 1",
		"francesouth":        "Zone 1",
		"australiacentral":   "Zone 4",
		"australiacentral2":  "Zone 4",
		"uaecentral":         "Zone 1",
		"uaenorth":           "Zone 1",
		"southafricanorth":   "Zone 1",
		"southafricawest":    "Zone 1",
		"switzerlandnorth":   "Zone 1",
		"switzerlandwest":    "Zone 1",
		"germanynorth":       "Zone 1",
		"germanywestcentral": "Zone 1",
		"norwayeast":         "Zone 1",
		"norwaywest":         "Zone 1",
		"brazilsoutheast":    "Zone 3",
		"westus3":            "Zone 1",
		"eastusslv":          "Zone 1",
		"swedencentral":      "Zone 1",
		"swedensouth":        "Zone 1",
	}[region]
}
