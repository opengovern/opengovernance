package cost_estimator

import (
	"context"
	"encoding/json"
	azureCompute "github.com/kaytu-io/kaytu-azure-describer/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-engine/pkg/describe/es"
)

// TODO we don't need to this file it will be deleted

func GetAzureResource(h *HttpHandler, resourceId string) (azureCompute.ComputeVirtualMachine, error) {
	// TODO we should add resourceType as another filter parameter
	query := map[string]interface{}{
		"bool": map[string]interface{}{
			"filter": map[string]interface{}{
				"term": map[string]interface{}{
					"ID": resourceId,
				},
			},
		},
	}

	queryJ, err := json.Marshal(query)
	if err != nil {
		return azureCompute.ComputeVirtualMachine{}, err
	}

	var response azureCompute.ComputeVirtualMachine
	err = h.client.Search(context.Background(), es.ResourceTypeToESIndex("Azure"), string(queryJ), response)
	if err != nil {
		return azureCompute.ComputeVirtualMachine{}, err
	}

	return response, nil
}

func GetAWSResource(h *HttpHandler, resourceId string) (azureCompute.ComputeVirtualMachine, error) {
	// TODO we should add resourceType as another filter parameter
	query := map[string]interface{}{
		"bool": map[string]interface{}{
			"filter": map[string]interface{}{
				"term": map[string]interface{}{
					"ID": resourceId,
				},
			},
		},
	}

	queryJ, err := json.Marshal(query)
	if err != nil {
		return azureCompute.ComputeVirtualMachine{}, err
	}

	var response azureCompute.ComputeVirtualMachine
	err = h.client.Search(context.Background(), es.ResourceTypeToESIndex("AWS"), string(queryJ), response)
	if err != nil {
		return azureCompute.ComputeVirtualMachine{}, err
	}

	return response, nil
}
