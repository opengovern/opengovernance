package inventory

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-05-01/network"
	ec2 "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	elasticsearchv7 "github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/google/uuid"
	awsdescriber "gitlab.com/keibiengine/keibi-engine/pkg/aws/describer"
	awsmodel "gitlab.com/keibiengine/keibi-engine/pkg/aws/model"
	azuredescriber "gitlab.com/keibiengine/keibi-engine/pkg/azure/describer"
	azuremodel "gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
	compliance_report "gitlab.com/keibiengine/keibi-engine/pkg/compliance-report"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/api"
)

func PopulateElastic(address string) error {
	cfg := elasticsearchv7.Config{
		Addresses: []string{address},
		Username:  "",
		Password:  "",
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, //nolint,gosec
			},
		},
	}

	es, err := elasticsearchv7.NewClient(cfg)
	if err != nil {
		return err
	}

	c, err := ioutil.ReadFile("test/compliance_report_template.json")
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", address+"/_index_template/compliance_report_template", bytes.NewReader(c))
	if err != nil {
		return err
	}
	req.Header.Add("Content-type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		return errors.New("invalid status code")
	}

	resources := GenerateLookupResources()
	for _, resource := range resources {
		err := IndexLookupResource(es, resource)
		if err != nil {
			return err
		}
	}

	err = GenerateResources(es)
	if err != nil {
		return err
	}

	u, err := uuid.Parse("2a87b978-b8bf-4d7e-bc19-cf0a99a430cf")
	if err != nil {
		return err
	}

	err = GenerateComplianceReport(es, u)
	if err != nil {
		return err
	}

	return nil
}

func GenerateLookupResources() []describe.KafkaLookupResource {
	sourceTypes := []string{"AWS", "AWS", "Azure", "Azure"}
	names := []string{"0001", "0002", "0003", "0004"}
	resourceIds := []string{"aaa0", "aaa1", "aaa2", "aaa3"}
	resourceTypes := []string{"AWS::EC2::Instance", "AWS::EC2::Instance", "Microsoft.Network/virtualNetworks", "Microsoft.Network/virtualNetworks"}
	resourceGroups := []string{"AA", "AB", "BA", "BB"}
	locations := []string{"us-east1", "us-east2", "us-east1", "us-east2"}
	sourceIDs := []string{"ss1", "ss1", "ss2", "ss2"}

	var resources []describe.KafkaLookupResource
	for i := 0; i < len(resourceIds); i++ {
		resource := describe.KafkaLookupResource{
			ResourceID:    resourceIds[i],
			Name:          names[i],
			SourceType:    api.SourceType(sourceTypes[i]),
			ResourceType:  resourceTypes[i],
			ResourceGroup: resourceGroups[i],
			Location:      locations[i],
			SourceID:      sourceIDs[i],
		}
		resources = append(resources, resource)
	}
	return resources
}

func IndexLookupResource(es *elasticsearchv7.Client, resource describe.KafkaLookupResource) error {
	js, err := json.Marshal(resource)
	if err != nil {
		return err
	}

	h := sha256.New()
	h.Write([]byte(resource.ResourceID))
	h.Write([]byte(resource.SourceType))
	documentID := fmt.Sprintf("%x", h.Sum(nil))

	// Set up the request object.
	req := esapi.IndexRequest{
		Index:      "inventory_summary",
		DocumentID: documentID,
		Body:       bytes.NewReader(js),
		Refresh:    "true",
	}

	// Perform the request with the client.
	res, err := req.Do(context.Background(), es)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("[%s] Error indexing document ID=%s", res.Status(), documentID)
	} else {
		// Deserialize the response into a map.
		var r map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
			return err
		}
	}
	return nil
}

func PopulatePostgres(db Database) error {
	err := db.AddQuery(&SmartQuery{
		Provider:    "AWS",
		Title:       "Query 1",
		Description: "description 1",
		Query:       "select count(*) from aws_ec2_instance",
	})
	if err != nil {
		return err
	}

	err = db.AddQuery(&SmartQuery{
		Provider:    "Azure",
		Title:       "Query 2",
		Description: "description 2",
		Query:       "select count(*) from azure_virtual_network",
	})
	if err != nil {
		return err
	}

	err = db.AddQuery(&SmartQuery{
		Provider:    "Azure",
		Title:       "Query 3",
		Description: "description 3",
		Query:       "select * from azure_virtual_network",
	})
	if err != nil {
		return err
	}

	err = db.AddQuery(&SmartQuery{
		Provider:    "AWS",
		Title:       "Query 4",
		Description: "description 4",
		Query:       "select * from aws_ec2_instance",
	})
	if err != nil {
		return err
	}

	err = db.AddBenchmark(&Benchmark{
		ID:          "test_compliance.benchmark1",
		Title:       "Benchmark 1",
		Description: "this is a benchmark",
		Provider:    "AWS",
		Tags: []BenchmarkTag{
			{
				Key:   "tagKey",
				Value: "tagValue",
			},
		},
		Policies: []Policy{
			{
				ID:                    "test_compliance.benchmark1.policy1",
				Title:                 "Policy 1",
				Description:           "description of policy 1",
				Tags:                  []PolicyTag{},
				Provider:              "AWS",
				Category:              "category1",
				SubCategory:           "sub_category1",
				Section:               "section1",
				Severity:              "high",
				ManualVerification:    "step1",
				ManualRemedation:      "step2",
				CommandLineRemedation: "step3",
				QueryToRun:            "query",
				KeibiManaged:          true,
			},
		},
	})
	if err != nil {
		return err
	}

	err = db.AddBenchmark(&Benchmark{
		ID:          "mod.azure_compliance",
		Title:       "Benchmark 2",
		Description: "this is another benchmark",
		Provider:    "Azure",
		Tags: []BenchmarkTag{
			{
				Key:   "tagKey",
				Value: "tagValue",
			},
			{
				Key:   "tag1",
				Value: "val1",
			},
		},
		Policies: []Policy{
			{
				ID:                    "control.cis_v130_1_21",
				Title:                 "Policy 2",
				Description:           "description of policy 2",
				Tags:                  []PolicyTag{},
				Provider:              "Azure",
				Category:              "category2",
				SubCategory:           "sub_category2",
				Section:               "section2",
				Severity:              "high",
				ManualVerification:    "step1",
				ManualRemedation:      "step2",
				CommandLineRemedation: "step3",
				QueryToRun:            "query",
				KeibiManaged:          true,
			},
			{
				ID:                    "control.cis_v130_1_23",
				Title:                 "Policy 3",
				Description:           "description of policy 3",
				Tags:                  []PolicyTag{},
				Provider:              "Azure",
				Category:              "category3",
				SubCategory:           "sub_category3",
				Section:               "section3",
				Severity:              "warn",
				ManualVerification:    "step1",
				ManualRemedation:      "step2",
				CommandLineRemedation: "step3",
				QueryToRun:            "query",
				KeibiManaged:          true,
			},
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func GenerateComplianceReport(es *elasticsearchv7.Client, sourceId uuid.UUID) error {
	r, err := compliance_report.ParseReport(
		"test/result-964df7ca-3ba4-48b6-a695-1ed9db5723f8-1645119195.json",
		1020,
		sourceId,
	)
	if err != nil {
		return err
	}

	for _, re := range r {
		b, err := json.Marshal(re)
		if err != nil {
			return err
		}

		u, err := uuid.NewUUID()
		if err != nil {
			return err
		}
		documentID := u.String()
		// Set up the request object.
		req := esapi.IndexRequest{
			Index:      compliance_report.ComplianceReportIndex,
			DocumentID: documentID,
			Body:       bytes.NewReader(b),
			Refresh:    "true",
		}

		// Perform the request with the client.
		res, err := req.Do(context.Background(), es)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		if res.IsError() {
			return fmt.Errorf("[%s] Error indexing document ID=%s", res.Status(), documentID)
		} else {
			// Deserialize the response into a map.
			var r map[string]interface{}
			if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
				return err
			}
		}
	}

	return nil
}

func BuildTempSpecFile(plugin string, esUrl string) (string, error) {
	spcFile, err := ioutil.TempFile("", plugin+"*.spc")
	if err != nil {
		return "", err
	}

	err = os.Chmod(spcFile.Name(), os.ModePerm)
	if err != nil {
		return spcFile.Name(), err
	}

	str := `
connection "` + plugin + `" {
  plugin = "` + plugin + `"
  addresses = ["` + esUrl + `"]
  username = ""
  password = ""
  accountID = "all"
}
`
	err = ioutil.WriteFile(spcFile.Name(), []byte(str), os.ModePerm)
	if err != nil {
		return spcFile.Name(), err
	}

	return spcFile.Name(), nil
}

func GenerateResources(es *elasticsearchv7.Client) error {
	instanceId := "abcd"
	empty := ""
	resource := awsdescriber.Resource{
		ARN: "abcd",
		ID:  "aaa0",
		Description: awsmodel.EC2InstanceDescription{
			Instance: &ec2.Instance{
				InstanceId:            &instanceId,
				StateTransitionReason: &empty,
				Tags:                  nil,
			},
			InstanceStatus: nil,
			Attributes: struct {
				UserData                          string
				InstanceInitiatedShutdownBehavior string
				DisableApiTermination             bool
			}{},
		},
		Name:      "0001",
		Account:   "ss1",
		Region:    "us-east1",
		Partition: "ppp",
		Type:      "AWS::EC2::Instance",
	}

	err := IndexAWSResource(es, resource)
	if err != nil {
		return err
	}

	azureResource := azuredescriber.Resource{
		ID: "aaa1",
		Description: azuremodel.VirtualNetworkDescription{
			VirtualNetwork: network.VirtualNetwork{
				VirtualNetworkPropertiesFormat: nil,
				Etag:                           nil,
				ID:                             nil,
				Name:                           nil,
				Type:                           nil,
				Location:                       nil,
				Tags:                           nil,
			},
			ResourceGroup: "abcd",
		},
		Name:           "0002",
		Type:           "Microsoft.Network/virtualNetworks",
		ResourceGroup:  "abcd",
		Location:       "us-east2",
		SubscriptionID: "ss2",
	}

	err = IndexAzureResource(es, azureResource)
	if err != nil {
		return err
	}

	azureResource = azuredescriber.Resource{
		ID: "aaa2",
		Description: azuremodel.VirtualNetworkDescription{
			VirtualNetwork: network.VirtualNetwork{
				VirtualNetworkPropertiesFormat: nil,
				Etag:                           nil,
				ID:                             nil,
				Name:                           nil,
				Type:                           nil,
				Location:                       nil,
				Tags:                           nil,
			},
			ResourceGroup: "abcd",
		},
		Name:           "0003",
		Type:           "Microsoft.Network/virtualNetworks",
		ResourceGroup:  "abcd",
		Location:       "us-east1",
		SubscriptionID: "ss1",
	}

	return IndexAzureResource(es, azureResource)
}

func IndexAWSResource(es *elasticsearchv7.Client, resource awsdescriber.Resource) error {
	kafkaRes := describe.KafkaResource{
		ID:            resource.UniqueID(),
		Description:   resource.Description,
		SourceType:    api.SourceCloudAWS,
		ResourceType:  resource.Type,
		ResourceJobID: uint(rand.Uint32()),
		SourceJobID:   uint(rand.Uint32()),
		Metadata: map[string]string{
			"partition":  resource.Partition,
			"region":     resource.Region,
			"account_id": resource.Account,
		},
	}
	return IndexKafkaResource(es, kafkaRes)
}

func IndexAzureResource(es *elasticsearchv7.Client, resource azuredescriber.Resource) error {
	kafkaRes := describe.KafkaResource{
		ID:            resource.UniqueID(),
		Description:   resource.Description,
		SourceType:    api.SourceCloudAzure,
		ResourceType:  resource.Type,
		ResourceJobID: uint(rand.Uint32()),
		SourceJobID:   uint(rand.Uint32()),
		Metadata: map[string]string{
			"id":                resource.ID,
			"name":              resource.Name,
			"subscription_id":   resource.SubscriptionID,
			"location":          resource.Location,
			"cloud_environment": "Azure",
		},
	}
	return IndexKafkaResource(es, kafkaRes)
}

func IndexKafkaResource(es *elasticsearchv7.Client, kafkaRes describe.KafkaResource) error {
	js, err := json.Marshal(kafkaRes)
	if err != nil {
		return err
	}

	h := sha256.New()
	h.Write([]byte(kafkaRes.ID))
	documentID := fmt.Sprintf("%x", h.Sum(nil))

	// Set up the request object.
	req := esapi.IndexRequest{
		Index:      describe.ResourceTypeToESIndex(kafkaRes.ResourceType),
		DocumentID: documentID,
		Body:       bytes.NewReader(js),
		Refresh:    "true",
	}

	// Perform the request with the client.
	res, err := req.Do(context.Background(), es)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("[%s] Error indexing document ID=%s", res.Status(), documentID)
	} else {
		// Deserialize the response into a map.
		var r map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
			return err
		}
	}
	return nil
}

type DescribeMock struct {
	Response []describe.ComplianceReportJob
}

func (m *DescribeMock) HelloServer(w http.ResponseWriter, r *http.Request) {
	var res []describe.ComplianceReportJob
	if r.URL.Query().Has("from") {
		fromStr := r.URL.Query().Get("from")
		toStr := r.URL.Query().Get("to")
		from, _ := strconv.ParseInt(fromStr, 10, 64)
		to, _ := strconv.ParseInt(toStr, 10, 64)
		for _, r := range m.Response {
			if r.Model.UpdatedAt.After(time.UnixMilli(from)) &&
				r.Model.UpdatedAt.Before(time.UnixMilli(to)) {
				res = append(res, r)
			}
		}
	} else {
		res = append(res, m.Response[len(m.Response)-1])
	}

	b, err := json.Marshal(res)
	if err != nil {
		fmt.Printf("Failed marshaling json: %v\n", err.Error())
	}

	_, err = fmt.Fprintf(w, string(b))
	if err != nil {
		fmt.Printf("Failed writing to response: %v\n", err.Error())
	}
}

func (m *DescribeMock) SetResponse(jobs ...describe.ComplianceReportJob) {
	m.Response = jobs
}

func (m *DescribeMock) Run() {
	http.HandleFunc("/api/v1/sources/", m.HelloServer)
	go http.ListenAndServe(":1234", nil)
	time.Sleep(100 * time.Millisecond)
}
