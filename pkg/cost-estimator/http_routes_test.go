package cost_estimator

import (
	"bytes"
	_ "context"
	"encoding/json"
	"fmt"
	_ "fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v4"
	elasticsearchv7 "github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/kaytu-io/kaytu-azure-describer/azure/model"
	azureCompute "github.com/kaytu-io/kaytu-azure-describer/pkg/kaytu-es-sdk"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

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

//	Things that need to be got from search elastics with the resourceId :{
//			ostype : VirtualMachine.prapertic.StorageProfile.OSDisk.OSType
//			location: VirtualMachine.Location
//			VMSize : VirtualMachine.prapertic.HardwareProfile.vmsize
//	}

func TestCreateAzureElastic(t *testing.T) {
	esCnfig := elasticsearch.Config{Addresses: []string{
		"http://localhost:9200",
	},
	}

	es, err := elasticsearch.NewClient(esCnfig)
	if err != nil {
		t.Errorf("error creating the client : %v ", err)
	}

	var reqEm azureCompute.ComputeVirtualMachine
	requestM, _ := json.Marshal(reqEm)
	req := bytes.NewBuffer(requestM)

	res, err := es.Create("azure", "2", req)
	if err != nil {
		t.Errorf("error in create the elastic search : %v ", err)
	}
	defer res.Body.Close()

	fmt.Println(res.String())
}

func TestSetAzureElastic(t *testing.T) {
	esCnfig := elasticsearchv7.Config{Addresses: []string{
		"http://localhost:9200",
	},
	}

	es, err := elasticsearchv7.NewClient(esCnfig)
	if err != nil {
		t.Errorf("error creating the client : %v ", err)
	}
	location := "eastus"
	vmSize := armcompute.VirtualMachineSizeTypesStandardDS2V2
	osType := armcompute.OperatingSystemTypeLinux
	virtualMachine := armcompute.VirtualMachine{
		Properties: &armcompute.VirtualMachineProperties{
			StorageProfile:  &armcompute.StorageProfile{OSDisk: &armcompute.OSDisk{OSType: (*armcompute.OperatingSystemTypes)(&osType)}},
			HardwareProfile: &armcompute.HardwareProfile{VMSize: &vmSize},
		},
		Location: &location,
	}
	var request = azureCompute.ComputeVirtualMachine{
		ResourceJobID: 1231,
		Description:   model.ComputeVirtualMachineDescription{VirtualMachine: virtualMachine},
	}

	requestM, _ := json.Marshal(request)
	bf := bytes.NewBuffer(requestM)

	res, err := es.Index("aws", bf)
	if err != nil {
		t.Errorf(err.Error())
	}
	defer res.Body.Close()

	fmt.Println(res.String())
}

func TestGetAzureElastic(t *testing.T) {
	esCnfig := elasticsearch.Config{Addresses: []string{
		"http://localhost:9200",
	},
	}

	es, err := elasticsearch.NewClient(esCnfig)
	if err != nil {
		t.Errorf("error creating the client : %v ", err)
	}

	res, err := es.Get("azure", "2")
	if err != nil {
		t.Errorf(err.Error())
	}

	defer res.Body.Close()
	fmt.Println(res.String())
	//var mapResp map[string]interface{}
	//if err := json.NewDecoder(res.Body).Decode(&mapResp); err != nil {
	//	t.Errorf(err.Error())
	//}
	//
	//fmt.Printf("mapResp TYPE: %v \n", reflect.TypeOf(mapResp))
	//for _, hit := range mapResp["hits"].(map[string]interface{})["hits"].([]interface{}) {
	//	// Parse the attributes/fields of the document
	//	doc := hit.(map[string]interface{})
	//
	//	// The "_source" data is another map interface nested inside of doc
	//	source := doc["_source"]
	//	fmt.Println("doc _source:", reflect.TypeOf(source))
	//
	//	// Get the document's _id and print it out along with _source data
	//	docID := doc["_id"]
	//	fmt.Println("docID:", docID)
	//	fmt.Println("_source:", source)
	//} // end of response iteration
}
