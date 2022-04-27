package inventory

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"time"

	"gorm.io/gorm"

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

func PopulateElastic(address string, d *DescribeMock) error {
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

	err = ApplyTemplate(address, "compliance_report_template")
	if err != nil {
		return err
	}

	err = ApplyTemplate(address, "account_report_template")
	if err != nil {
		return err
	}
	err = ApplyTemplate(address, "resource_growth_template")
	if err != nil {
		return err
	}

	resources := GenerateLookupResources()
	for _, resource := range resources {
		err := IndexKafkaMessage(es, resource)
		if err != nil {
			return err
		}
	}

	for _, resource := range GenerateResourceGrowth() {
		err := IndexKafkaMessage(es, resource)
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

	createdAt := time.Now().UnixMilli()
	j1 := describe.ComplianceReportJob{
		Model: gorm.Model{
			ID:        1,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			DeletedAt: gorm.DeletedAt{},
		},
		SourceID:        u,
		Status:          compliance_report.ComplianceReportJobCompleted,
		ReportCreatedAt: createdAt,
		FailureMessage:  "",
	}
	d.SetResponse(j1)

	err = GenerateAccountReport(es, u, 1020, createdAt)
	if err != nil {
		return err
	}

	err = GenerateComplianceReport(es, u, 1020, createdAt)
	if err != nil {
		return err
	}

	createdAt = time.Now().UnixMilli()
	err = GenerateComplianceReport(es, u, 1023, createdAt)
	if err != nil {
		return err
	}

	return nil
}

func ApplyTemplate(address string, templateName string) error {
	c, err := ioutil.ReadFile("test/" + templateName + ".json")
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", address+"/_index_template/"+templateName, bytes.NewReader(c))
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

func GenerateResourceGrowth() []describe.KafkaResourceGrowthTrendResource {
	var resources []describe.KafkaResourceGrowthTrendResource
	for i := 0; i < 4; i++ {
		resource := describe.KafkaResourceGrowthTrendResource{
			SourceID:      "2a87b978-b8bf-4d7e-bc19-cf0a99a430cf",
			SourceType:    "AWS",
			SourceJobID:   1020,
			CreatedAt:     time.Now().UnixMilli(),
			ResourceCount: i * 10,
		}
		resources = append(resources, resource)
	}
	return resources
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
		Tags: []Tag{
			{
				Value: "tag1",
			},
		},
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
				ID:                    "control.cis_v130_7_1",
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

func GenerateAccountReport(es *elasticsearchv7.Client, sourceId uuid.UUID, jobID uint, createdAt int64) error {
	r := compliance_report.AccountReport{
		SourceID:    sourceId,
		BenchmarkID: "azure_compliance.benchmark.cis_v130",
		ReportJobId: jobID,
		Summary: compliance_report.Summary{
			Status: compliance_report.SummaryStatus{
				Alarm: 0,
				OK:    20,
				Info:  0,
				Skip:  1,
				Error: 0,
			},
		},
		CreatedAt:            createdAt,
		TotalResources:       21,
		TotalCompliant:       20,
		CompliancePercentage: 0.99,
	}
	return IndexKafkaMessage(es, r)
}
func GenerateComplianceReport(es *elasticsearchv7.Client, sourceId uuid.UUID, jobID uint, createdAt int64) error {
	r, err := compliance_report.ParseReport(
		"test/result-964df7ca-3ba4-48b6-a695-1ed9db5723f8-1645119195.json",
		jobID,
		sourceId,
		createdAt,
		compliance_report.SourceCloudAzure,
	)
	if err != nil {
		return err
	}

	for _, re := range r {
		err = IndexKafkaMessage(es, re)
		if err != nil {
			return err
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
	return IndexKafkaMessage(es, kafkaRes)
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
	return IndexKafkaMessage(es, kafkaRes)
}

func IndexKafkaMessage(es *elasticsearchv7.Client, kafkaRes describe.KafkaMessage) error {
	r, err := kafkaRes.AsProducerMessage()
	if err != nil {
		return err
	}

	id, err := r.Key.Encode()
	if err != nil {
		return err
	}

	body, err := r.Value.Encode()
	if err != nil {
		return err
	}

	var index string
	for _, header := range r.Headers {
		if string(header.Key) == "elasticsearch_index" {
			index = string(header.Value)
		}
	}

	// Set up the request object.
	req := esapi.IndexRequest{
		Index:      index,
		DocumentID: string(id),
		Body:       bytes.NewReader(body),
		Refresh:    "true",
	}

	// Perform the request with the client.
	res, err := req.Do(context.Background(), es)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("[%s] Error indexing document ID=%s", res.Status(), string(id))
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
	Response   []describe.ComplianceReportJob
	MockServer *httptest.Server
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
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/sources/", m.HelloServer)
	m.MockServer = httptest.NewServer(mux)
}
