package azure

import (
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/api"
	"github.com/kaytu-io/kaytu-engine/pkg/workspace/costestimator/query"
	"go.uber.org/zap"
)

var (
	locationDisplayToName = map[string]string{
		"West US":              "westus",
		"West US 2":            "westus2",
		"East US":              "eastus",
		"Central US":           "centralus",
		"Central US EUAP":      "centraluseuap",
		"South Central US":     "southcentralus",
		"North Central US":     "northcentralus",
		"West Central US":      "westcentralus",
		"East US 2":            "eastus2",
		"East US 2 EUAP":       "eastus2euap",
		"Brazil South":         "brazilsouth",
		"Brazil US":            "brazilus",
		"North Europe":         "northeurope",
		"West Europe":          "westeurope",
		"East Asia":            "eastasia",
		"Southeast Asia":       "southeastasia",
		"Japan West":           "japanwest",
		"Japan East":           "japaneast",
		"Korea Central":        "koreacentral",
		"Korea South":          "koreasouth",
		"South India":          "southindia",
		"West India":           "westindia",
		"Central India":        "centralindia",
		"Australia East":       "australiaeast",
		"Australia Southeast":  "australiasoutheast",
		"Canada Central":       "canadacentral",
		"Canada East":          "canadaeast",
		"UK South":             "uksouth",
		"UK West":              "ukwest",
		"France Central":       "francecentral",
		"France South":         "francesouth",
		"Australia Central":    "australiacentral",
		"Australia Central 2":  "australiacentral2",
		"UAE Central":          "uaecentral",
		"UAE North":            "uaenorth",
		"South Africa North":   "southafricanorth",
		"South Africa West":    "southafricawest",
		"Switzerland North":    "switzerlandnorth",
		"Switzerland West":     "switzerlandwest",
		"Germany North":        "germanynorth",
		"Germany West Central": "germanywestcentral",
		"Norway East":          "norwayeast",
		"Norway West":          "norwaywest",
		"Brazil Southeast":     "brazilsoutheast",
		"West US 3":            "westus3",
		"East US SLV":          "eastusslv",
		"Sweden Central":       "swedencentral",
		"Sweden South":         "swedensouth",
	}
)

// Provider is an implementation of the terraform.Provider, used to extract component queries from
// terraform resources.
type Provider struct {
	key string
}

// NewProvider initializes a new provider with key
func NewProvider(key string) (*Provider, error) {
	return &Provider{
		key: key,
	}, nil
}

// Name returns the Provider's common name.
func (p *Provider) Name() string { return p.key }

// ResourceComponents returns Component queries for a given terraform.Resource.
func (p *Provider) ResourceComponents(logger *zap.Logger, resourceType string, request any) ([]query.Component, error) {
	fmt.Println("REQUEST: ", request)
	fmt.Println("RESOURCE TYPE: ", resourceType)
	switch resourceType {
	case "azurerm_linux_virtual_machine":
		var vmRequest api.GetAzureVmRequest
		if req, ok := request.(map[string]interface{}); ok {
			vmRequest = api.GetAzureVmRequest{
				RegionCode: req["RegionCode"].(string),
				VMSize:     req["VMSize"].(string),
			}
		} else {
			return nil, fmt.Errorf("could not parse request")
		}
		vals := decodeLinuxVirtualMachineValues(vmRequest)
		return p.newLinuxVirtualMachine(vals).Components(), nil
	case "azurerm_virtual_machine":
		var vmRequest api.GetAzureVmRequest
		if req, ok := request.(map[string]interface{}); ok {
			vmRequest = api.GetAzureVmRequest{
				RegionCode:      req["RegionCode"].(string),
				VMSize:          req["VMSize"].(string),
				OperatingSystem: req["OperatingSystem"].(string),
			}
		} else {
			return nil, fmt.Errorf("could not parse request")
		}
		vals := decodeVirtualMachineValues(vmRequest)
		return p.newVirtualMachine(vals).Components(), nil
	case "azurerm_managed_disk":
		var mdRequest api.GetAzureManagedStorageRequest
		if req, ok := request.(map[string]interface{}); ok {
			mdRequest = api.GetAzureManagedStorageRequest{
				RegionCode:      req["RegionCode"].(string),
				SkuName:         req["SkuName"].(string),
				DiskSize:        req["DiskSize"].(float64),
				BurstingEnabled: req["BurstingEnabled"].(bool),
				DiskThroughput:  req["DiskThroughput"].(float64),
				DiskIOPs:        req["DiskIOPs"].(float64),
			}
		} else {
			return nil, fmt.Errorf("could not parse request")
		}
		vals := decodeManagedStorageValues(mdRequest)
		logger.Info("Vals", zap.Any("Vals", vals))
		fmt.Println("Vals: ", vals)
		return p.newManagedStorage(vals).Components(), nil
	case "azurerm_sql_server_DB":
		var sqlSDB api.GetAzureSqlServersDatabasesRequest
		if req, ok := request.(api.GetAzureSqlServersDatabasesRequest); ok {
			sqlSDB = req
		} else {
			return nil, fmt.Errorf("could not parse request")
		}
		vals := decodeSqlServerDB(sqlSDB, sqlSDB.MonthlyVCoreHours, sqlSDB.ExtraDataStorageGB, sqlSDB.LongTermRetentionStorageGB, sqlSDB.BackupStorageGB)
		return p.newSqlServerDB(vals).Components(), nil
	case "azurerm_load_balancer":
		var lbRequest api.GetAzureLoadBalancerRequest
		if req, ok := request.(map[string]interface{}); ok {
			lbRequest = api.GetAzureLoadBalancerRequest{
				RegionCode:       req["RegionCode"].(string),
				DailyDataProceed: nil,
				SkuName:          req["SkuName"].(string),
				SkuTier:          req["SkuTier"].(string),
				RulesNumber:      req["RulesNumber"].(int32),
			}
		} else {
			return nil, fmt.Errorf("could not parse request")
		}
		vals := decodeLoadBalancerValues(lbRequest)
		logger.Info("Vals", zap.Any("Vals", vals))
		return p.newLoadBalancer(vals).Components(), nil
	case "azurerm_virtual_network":
		var vnRequest api.GetAzureVirtualNetworkRequest
		if req, ok := request.(map[string]interface{}); ok {
			vnRequest = api.GetAzureVirtualNetworkRequest{
				RegionCode:            req["RegionCode"].(string),
				PeeringLocations:      req["PeeringLocations"].([]string),
				MonthlyDataTransferGB: nil,
			}
		} else {
			return nil, fmt.Errorf("could not parse request")
		}
		vals := decodeVirtualNetworkValues(vnRequest)
		logger.Info("Vals", zap.Any("Vals", vals))
		return p.newVirtualNetwork(vals).Components(), nil
	default:
		return nil, fmt.Errorf("resource type not found in cost calculator")
	}
}

// getLocationName will return the location name from the location display name (ex: UK West -> ukwest)
// if the l is not found it'll return the l again meaning is not found or already a name
func getLocationName(l string) string {
	ln, ok := locationDisplayToName[l]
	if !ok {
		return l
	}
	return ln
}
