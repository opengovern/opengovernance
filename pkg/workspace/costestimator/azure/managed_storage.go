package azure

// https://azure.microsoft.com/en-us/pricing/details/managed-disks/#resources

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"strings"
)

type skuDetails struct {
	DiskOption string
	DiskSize   int32
}

func ManagedStorageCostByResource(db *db.Database, request api.GetAzureManagedStorageRequest) (float64, error) {
	var cost float64
	sku := strings.Split(string(*request.ManagedStorage.Disk.SKU.Name), "_")
	if sku[0] == "PremiumV2" {
		return 0, fmt.Errorf("PremiumV2 disk not supported") // This is calculated per Gigabytes
	}
	if sku[0] == "UltraSSD" {
		throughputPrice, err := db.FindAzureManagedStoragePrice(request.RegionCode, "Ultra LRS", "Ultra LRS Provisioned Throughput (MBps)")
		if err != nil {
			return 0, err
		}
		cost += throughputPrice.Price * float64(*request.ManagedStorage.Disk.Properties.DiskMBpsReadWrite)
		capacityPrice, err := db.FindAzureManagedStoragePrice(request.RegionCode, "Ultra LRS", "Ultra LRS Provisioned Capacity")
		if err != nil {
			return 0, err
		}
		cost += capacityPrice.Price * float64(*request.ManagedStorage.Disk.Properties.DiskSizeGB)
		iopsPrice, err := db.FindAzureManagedStoragePrice(request.RegionCode, "Ultra LRS", "Ultra LRS Provisioned IOPS")
		if err != nil {
			return 0, err
		}

		// TODO: Take care of vCPU reservation
		// Provisioned vcpu reservation charge :: This reservation charge is only imposed if you enable Ultra Disk compatibility on the VM without attaching an Ultra Disk.

		cost += iopsPrice.Price * float64(*request.ManagedStorage.Disk.Properties.DiskIOPSReadWrite)
	} else {
		skuName := skuNames[skuDetails{DiskOption: sku[0], DiskSize: *request.ManagedStorage.Disk.Properties.DiskSizeGB}]
		skuName = fmt.Sprintf("%s %s", skuName, sku[1])
		price, err := db.FindAzureManagedStoragePrice(request.RegionCode, skuName, "Per Month")
		if err != nil {
			return 0, nil
		}
		numberOfDays := costestimator.GetNumberOfDays()
		cost += (price.Price / (float64(numberOfDays))) / 24
		if (sku[0] == "Premium") && (*request.ManagedStorage.Disk.Properties.DiskSizeGB >= 1024) && *request.ManagedStorage.Disk.Properties.BurstingEnabled {
			burstPrice, err := db.FindAzureManagedStoragePrice(request.RegionCode, "Burst Enablement LRS", "Per Month")
			if err != nil {
				return 0, nil
			}
			cost += (burstPrice.Price / (float64(numberOfDays))) / 24
		}
	}
	return cost * costestimator.TimeInterval, nil
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
