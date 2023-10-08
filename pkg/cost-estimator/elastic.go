package cost_estimator

import (
	"context"
	"encoding/json"
	azureCompute "github.com/kaytu-io/kaytu-azure-describer/pkg/kaytu-es-sdk"
)

func GetAzureResource(h *HttpHandler, resourceId string) (azureCompute.ComputeVirtualMachine, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"querytype": map[string]interface{}{
				"ResourceJobID": resourceId,
			},
		},
	}

	queryJ, err := json.Marshal(query)
	if err != nil {
		return azureCompute.ComputeVirtualMachine{}, err
	}

	var response azureCompute.ComputeVirtualMachine
	err = h.client.Search(context.Background(), "test", string(queryJ), response)
	if err != nil {
		return azureCompute.ComputeVirtualMachine{}, err
	}

	return response, nil
}
