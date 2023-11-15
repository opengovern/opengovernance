package resource_types

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/price"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/product"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/query"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/util"
	"github.com/shopspring/decimal"
	"strings"
)

// ManagedStorage is the entity that holds the logic to calculate price
// of the google_compute_instance
type ManagedStorage struct {
	provider *Provider

	location        string
	skuName         string
	diskSize        int32
	burstingEnabled bool
	diskThroughput  int64
	diskIOPs        int64
}

// managedStorageValues is holds the values that we need to be able
// to calculate the price of the ComputeInstance
type managedStorageValues struct {
	SkuName         string `mapstructure:"sku_name"`
	Location        string `mapstructure:"location"`
	DiskSize        int32  `mapstructure:"disk_size"`
	BurstingEnabled bool   `mapstructure:"bursting_enabled"`
	DiskThroughput  int64  `mapstructure:"disk_throughput"`
	DiskIOPs        int64  `mapstructure:"disk_iops"`
}

// decodeManagedStorageValues decodes and returns computeInstanceValues from a Terraform values map.
func decodeManagedStorageValues(request api.GetAzureManagedStorageRequest) managedStorageValues {
	return managedStorageValues{
		SkuName:         string(*request.ManagedStorage.Disk.SKU.Name),
		Location:        request.RegionCode,
		DiskSize:        *request.ManagedStorage.Disk.Properties.DiskSizeGB,
		BurstingEnabled: *request.ManagedStorage.Disk.Properties.BurstingEnabled,
		DiskThroughput:  *request.ManagedStorage.Disk.Properties.DiskMBpsReadWrite,
		DiskIOPs:        *request.ManagedStorage.Disk.Properties.DiskIOPSReadWrite,
	}
}

// newManagedStorage initializes a new VirtualMachine from the provider
func (p *Provider) newManagedStorage(vals managedStorageValues) *ManagedStorage {
	inst := &ManagedStorage{
		provider: p,

		location:        getLocationName(vals.Location),
		skuName:         vals.SkuName,
		diskSize:        vals.DiskSize,
		burstingEnabled: vals.BurstingEnabled,
		diskThroughput:  vals.DiskThroughput,
		diskIOPs:        vals.DiskIOPs,
	}

	return inst
}

// Components returns the price component queries that make up this Instance.
func (inst *ManagedStorage) Components() []query.Component {
	var components []query.Component

	sku := strings.Split(inst.skuName, "_")
	if sku[0] == "PremiumV2" {
		return nil // Not Supported
	} else if sku[0] == "UltraSSD" {
		components = append(components, inst.ultraLRSThroughputComponent(inst.provider.key, inst.location, inst.diskThroughput))
		components = append(components, inst.ultraLRSCapacityComponent(inst.provider.key, inst.location, inst.diskSize))
		components = append(components, inst.ultraLRSIOPsComponent(inst.provider.key, inst.location, inst.diskIOPs))
		// TODO: Take care of vCPU reservation
		// Provisioned vcpu reservation charge :: This reservation charge is only imposed if you enable Ultra Disk compatibility on the VM without attaching an Ultra Disk.
	} else {
		skuName := skuNames[skuDetails{DiskOption: sku[0], DiskSize: inst.diskSize}]
		skuName = fmt.Sprintf("%s %s", skuName, sku[1])
		components = []query.Component{inst.managedStorageComponent(inst.provider.key, inst.location, skuName)}

		if (sku[0] == "Premium") && (inst.diskSize >= 1024) && inst.burstingEnabled {
			components = append(components, inst.enableBurstingComponent(inst.provider.key, inst.location))
		}
	}
	return components
}

// ultraLRSThroughputComponent Throughput of Ultra LRS
func (inst *ManagedStorage) ultraLRSThroughputComponent(key, location string, throughput int64) query.Component {
	return query.Component{
		Name:           "Ultra LRS Throughput",
		HourlyQuantity: decimal.NewFromInt(throughput),
		ProductFilter: &product.Filter{
			Provider: util.StringPtr(key),
			Service:  util.StringPtr("Storage"),
			Family:   util.StringPtr("Storage"),
			Location: util.StringPtr(location),
			AttributeFilters: []*product.AttributeFilter{
				{Key: "sku_name", Value: util.StringPtr("Ultra LRS")},
				{Key: "meter", Value: util.StringPtr("Ultra LRS Provisioned Throughput (MBps)")},
			},
		},
	}
}

// ultraLRSCapacityComponent Capacity of Ultra LRS
func (inst *ManagedStorage) ultraLRSCapacityComponent(key, location string, diskSize int32) query.Component {
	return query.Component{
		Name:           "Ultra LRS Capacity",
		HourlyQuantity: decimal.NewFromInt(int64(diskSize)),
		ProductFilter: &product.Filter{
			Provider: util.StringPtr(key),
			Service:  util.StringPtr("Storage"),
			Family:   util.StringPtr("Storage"),
			Location: util.StringPtr(location),
			AttributeFilters: []*product.AttributeFilter{
				{Key: "sku_name", Value: util.StringPtr("Ultra LRS")},
				{Key: "meter", Value: util.StringPtr("Ultra LRS Provisioned Capacity")},
			},
		},
	}
}

// ultraLRSIOPsComponent IOPs for Ultra LRS
func (inst *ManagedStorage) ultraLRSIOPsComponent(key, location string, iops int64) query.Component {
	return query.Component{
		Name:           "Ultra LRS IOPs",
		HourlyQuantity: decimal.NewFromInt(iops),
		ProductFilter: &product.Filter{
			Provider: util.StringPtr(key),
			Service:  util.StringPtr("Storage"),
			Family:   util.StringPtr("Storage"),
			Location: util.StringPtr(location),
			AttributeFilters: []*product.AttributeFilter{
				{Key: "sku_name", Value: util.StringPtr("Ultra LRS")},
				{Key: "meter", Value: util.StringPtr("Ultra LRS Provisioned IOPS")},
			},
		},
	}
}

// managedStorageComponent is the component for Premium and Standard Managed Storages
func (inst *ManagedStorage) managedStorageComponent(key, location, skuName string) query.Component {
	return query.Component{
		Name:            "Managed Storage",
		MonthlyQuantity: decimal.NewFromInt(1),
		ProductFilter: &product.Filter{
			Provider: util.StringPtr(key),
			Service:  util.StringPtr("Storage"),
			Family:   util.StringPtr("Storage"),
			Location: util.StringPtr(location),
			AttributeFilters: []*product.AttributeFilter{
				{Key: "sku_name", Value: util.StringPtr(skuName)},
				{Key: "meter", Value: util.StringPtr("Per Month")},
			},
		},
		PriceFilter: &price.Filter{
			Unit: util.StringPtr("1 Hour"),
		},
	}
}

// enableBurstingComponent component for when the Bursting is enabled for the managed storage
func (inst *ManagedStorage) enableBurstingComponent(key, location string) query.Component {
	return query.Component{
		Name:            "Enable Bursting",
		MonthlyQuantity: decimal.NewFromInt(1),
		ProductFilter: &product.Filter{
			Provider: util.StringPtr(key),
			Service:  util.StringPtr("Storage"),
			Family:   util.StringPtr("Storage"),
			Location: util.StringPtr(location),
			AttributeFilters: []*product.AttributeFilter{
				{Key: "sku_name", Value: util.StringPtr("Burst Enablement LRS")},
				{Key: "meter", Value: util.StringPtr("Per Month")},
			},
		},
	}
}

type skuDetails struct {
	DiskOption string
	DiskSize   int32
}

var skuNames = map[skuDetails]string{
	skuDetails{DiskOption: "Standard", DiskSize: 32}:    "S4",
	skuDetails{DiskOption: "Standard", DiskSize: 64}:    "S6",
	skuDetails{DiskOption: "Standard", DiskSize: 128}:   "S10",
	skuDetails{DiskOption: "Standard", DiskSize: 256}:   "S15",
	skuDetails{DiskOption: "Standard", DiskSize: 512}:   "S20",
	skuDetails{DiskOption: "Standard", DiskSize: 1024}:  "S30",
	skuDetails{DiskOption: "Standard", DiskSize: 2048}:  "S40",
	skuDetails{DiskOption: "Standard", DiskSize: 4096}:  "S50",
	skuDetails{DiskOption: "Standard", DiskSize: 8192}:  "S60",
	skuDetails{DiskOption: "Standard", DiskSize: 16384}: "S70",
	skuDetails{DiskOption: "Standard", DiskSize: 32767}: "S80",

	skuDetails{DiskOption: "StandardSSD", DiskSize: 4}:     "E1",
	skuDetails{DiskOption: "StandardSSD", DiskSize: 8}:     "E2",
	skuDetails{DiskOption: "StandardSSD", DiskSize: 16}:    "E3",
	skuDetails{DiskOption: "StandardSSD", DiskSize: 32}:    "E4",
	skuDetails{DiskOption: "StandardSSD", DiskSize: 64}:    "E6",
	skuDetails{DiskOption: "StandardSSD", DiskSize: 128}:   "E10",
	skuDetails{DiskOption: "StandardSSD", DiskSize: 256}:   "E15",
	skuDetails{DiskOption: "StandardSSD", DiskSize: 512}:   "E20",
	skuDetails{DiskOption: "StandardSSD", DiskSize: 1024}:  "E30",
	skuDetails{DiskOption: "StandardSSD", DiskSize: 2048}:  "E40",
	skuDetails{DiskOption: "StandardSSD", DiskSize: 4096}:  "E50",
	skuDetails{DiskOption: "StandardSSD", DiskSize: 8192}:  "E60",
	skuDetails{DiskOption: "StandardSSD", DiskSize: 16384}: "E70",
	skuDetails{DiskOption: "StandardSSD", DiskSize: 32767}: "E80",

	skuDetails{DiskOption: "Premium", DiskSize: 4}:     "P1",
	skuDetails{DiskOption: "Premium", DiskSize: 8}:     "P2",
	skuDetails{DiskOption: "Premium", DiskSize: 16}:    "P3",
	skuDetails{DiskOption: "Premium", DiskSize: 32}:    "P4",
	skuDetails{DiskOption: "Premium", DiskSize: 64}:    "P6",
	skuDetails{DiskOption: "Premium", DiskSize: 128}:   "P10",
	skuDetails{DiskOption: "Premium", DiskSize: 256}:   "P15",
	skuDetails{DiskOption: "Premium", DiskSize: 512}:   "P20",
	skuDetails{DiskOption: "Premium", DiskSize: 1024}:  "P30",
	skuDetails{DiskOption: "Premium", DiskSize: 2048}:  "P40",
	skuDetails{DiskOption: "Premium", DiskSize: 4096}:  "P50",
	skuDetails{DiskOption: "Premium", DiskSize: 8192}:  "P60",
	skuDetails{DiskOption: "Premium", DiskSize: 16384}: "P70",
	skuDetails{DiskOption: "Premium", DiskSize: 32767}: "P80",
}
