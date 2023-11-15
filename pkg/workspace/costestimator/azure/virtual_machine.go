package azure

import (
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/db"
	"strings"
)

func VmCostByResource(db *db.Database, request api.GetAzureVmRequest) (float64, error) {
	var cost float64
	prices, err := db.FindAzureVMPrice(request.RegionCode, string(*request.VM.VirtualMachine.Properties.HardwareProfile.VMSize), "regular")
	if err != nil {
		return 0, nil
	}
	for _, p := range prices {
		if string(*request.VM.VirtualMachine.Properties.StorageProfile.OSDisk.OSType) == "Windows" {
			if strings.Contains(p.ProductName, "Windows") {
				cost += p.Price.InexactFloat64()
			}
		} else {
			if !strings.Contains(p.ProductName, "Windows") {
				cost += p.Price.InexactFloat64()
			}
		}
	}
	return cost * costestimator.TimeInterval, nil
}
