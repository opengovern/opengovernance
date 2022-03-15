package inventory

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"

	elasticsearchv7 "github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
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
				InsecureSkipVerify: true,
			},
		},
	}

	es, err := elasticsearchv7.NewClient(cfg)
	if err != nil {
		return err
	}

	resources := GenerateLookupResources()
	for _, resource := range resources {
		err := IndexLookupResource(es, resource)
		if err != nil {
			return err
		}
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
