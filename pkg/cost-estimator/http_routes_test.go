package cost_estimator

import (
	_ "context"
	"encoding/json"
	"fmt"
	_ "fmt"
	azure "github.com/kaytu-io/kaytu-azure-describer/azure/model"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type ComputeVirtualMachine struct {
	Description   azure.ComputeVirtualMachineDescription `json:"description"`
	Metadata      azure.Metadata                         `json:"metadata"`
	ResourceJobID int                                    `json:"resource_job_id"`
	SourceJobID   int                                    `json:"source_job_id"`
	ResourceType  string                                 `json:"resource_type"`
	SourceType    string                                 `json:"source_type"`
	ID            string                                 `json:"id"`
	ARN           string                                 `json:"arn"`
	SourceID      string                                 `json:"source_id"`
}

type ItemsStr struct {
	CurrencyCode         string
	TierMinimumUnits     float64
	RetailPrice          float64
	UnitPrice            float64
	ArmRegionName        string
	Location             string
	EffectiveStartDate   string
	MeterId              string
	MeterName            string
	ProductId            string
	SkuId                string
	ProductName          string
	SkuName              string
	ServiceName          string
	ServiceId            string
	ServiceFamily        string
	UnitOfMeasure        string
	Type                 string
	IsPrimaryMeterRegion bool
	ArmSkuName           string
}
type AzureCostStr struct {
	BillingCurrency    string
	CustomerEntityId   string
	CustomerEntityType string
	Items              []ItemsStr
	NextPageLink       string
	Count              int
}

//	the thinks that needs for giving from elastic search from each resource :{
//			ostype : VirtualMachine.prapertic.StorageProfile.OSDisk.OSType
//			location: VirtualMachine.Location
//			VMSize : VirtualMachine.prapertic.HardwareProfile.vmsize
//	}

func TestAzureCostRequest(t *testing.T) {
	serviceName := "Virtual Machines"
	OSType := "Windows"
	typeN := "Consumption"
	armRegionName := "eastus"
	serviceFamily := "Compute"
	armSkuName := "Standard_E16ds_v5"
	filter := fmt.Sprintf("serviceName eq '%v' and type eq '%v' and serviceFamily eq '%v' and armSkuName eq '%v' and armRegionName eq '%v' ", serviceName, typeN, serviceFamily, armSkuName, armRegionName)

	url := "https://prices.azure.com/api/retail/prices"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Errorf("error : %v ", err)
	}

	q := req.URL.Query()
	q.Add("$filter", filter)
	req.URL.RawQuery = q.Encode()

	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		t.Errorf("error in status code : %v ", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Errorf("error status equal to : %v ", res.StatusCode)
	}

	responseBody, err := io.ReadAll(res.Body)
	if err != nil {
		t.Errorf("error  : %v ", err)
	}

	var response AzureCostStr
	err = json.Unmarshal(responseBody, &response)
	if err != nil {
		t.Errorf("error in unmarshalling the response : %v ", err)
	}
	//fmt.Printf("items : %v ", response.Items)
	item := giveProperCostTime(response.Items, t, OSType)
	fmt.Printf("cost equal to : %v ", item.RetailPrice)
}

func giveProperCostTime(Items []ItemsStr, t *testing.T, OSType string) ItemsStr {
	newTime := 1
	var newItem ItemsStr
	osTypeCheckWindows := true
	if OSType == "Linux" {
		osTypeCheckWindows = false
	}

	for i := 0; i < len(Items); i++ {
		item := Items[i]

		checkOsType := strings.Contains(item.ProductName, "Windows")
		if osTypeCheckWindows {
			if !checkOsType {
				continue
			}
		} else {
			if checkOsType {
				continue
			}
		}

		timeP, err := time.Parse(time.RFC3339, item.EffectiveStartDate)
		if err != nil {
			t.Errorf("error in parsing time : %v ", err)
		}
		if timeP.Year() > newTime {
			newTime = timeP.Year()
			newItem = Items[i]
		}
	}
	return newItem
}
