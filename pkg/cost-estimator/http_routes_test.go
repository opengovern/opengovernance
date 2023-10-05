package cost_estimator

import (
	"bytes"
	_ "context"
	"encoding/json"
	"fmt"
	_ "fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v4"
	elastic "github.com/elastic/go-elasticsearch/v7"
	azure "github.com/kaytu-io/kaytu-azure-describer/azure/model"
	"net/http"
	"testing"
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

func TestAddNewResource(t *testing.T) {
	client, err := elastic.NewClient(elastic.Config{
		Addresses: []string{"http://localhost:9200"},
	})
	if err != nil {
		t.Errorf("error in connecting to elastic : %v ", err)
	}

	var location string = "IN South"
	body := ComputeVirtualMachine{
		SourceType:  "Virtual Machines",
		Description: azure.ComputeVirtualMachineDescription{VirtualMachine: armcompute.VirtualMachine{Location: &location}},
	}
	bodyM, err := json.Marshal(body)
	if err != nil {
		t.Errorf("err : %v ", err)
	}

	test := bytes.NewBuffer(bodyM)
	response, err := client.Create("new-resource", "1", test)
	if err != nil {
		t.Errorf("err : %v ", err)
	}

	if response.StatusCode != http.StatusOK {
		t.Errorf("err : %v ", response.StatusCode)
	}
	fmt.Println(response.Body)

}
