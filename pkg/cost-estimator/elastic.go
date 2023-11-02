package cost_estimator

import (
	"context"
	"encoding/json"
	azureCompute "github.com/kaytu-io/kaytu-azure-describer/pkg/kaytu-es-sdk"
	"github.com/kaytu-io/kaytu-util/pkg/es"
	"go.uber.org/zap"
)

func (h *HttpHandler) GetComputeVirtualMachine(resourceId string) (*azureCompute.ComputeVirtualMachine, error) {
	var resp azureCompute.ComputeVirtualMachine
	err := h.GetResource("Microsoft.Compute/virtualMachines", resourceId, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (h *HttpHandler) GetVirtualNetwork(resourceId string) (*azureCompute.VirtualNetwork, error) {
	var resp azureCompute.VirtualNetwork
	err := h.GetResource("Microsoft.Network/virtualNetworks", resourceId, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (h *HttpHandler) GetResource(resourceType string, resourceId string, resp any) error {

	index := es.ResourceTypeToESIndex(resourceType)

	terms := make(map[string]any)
	terms["id"] = resourceId

	root := map[string]any{}

	boolQuery := make(map[string]any)
	if terms != nil && len(terms) > 0 {
		var filters []map[string]any
		for k, vs := range terms {
			filters = append(filters, map[string]any{
				"terms": map[string]any{
					k: vs,
				},
			})
		}

		boolQuery["filter"] = filters
	}
	if len(boolQuery) > 0 {
		root["query"] = map[string]any{
			"bool": boolQuery,
		}
	}

	queryBytes, err := json.Marshal(root)
	if err != nil {
		return err
	}

	h.logger.Info("GetResource", zap.String("query", string(queryBytes)), zap.String("index", index))
	err = h.client.Search(context.Background(), index, string(queryBytes), resp)
	return err
}
