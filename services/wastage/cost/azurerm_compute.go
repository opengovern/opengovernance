package cost

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kaytu-io/kaytu-engine/pkg/httpclient"
	"github.com/kaytu-io/kaytu-engine/services/wastage/api/entity"
	"github.com/kaytu-io/pennywise/pkg/cost"
	"github.com/kaytu-io/pennywise/pkg/schema"
	"net/http"
	"time"
)

func (s *Service) GetAzureComputeVMCost(ctx context.Context, instance entity.AzureVM) (float64, error) {
	req := schema.Submission{
		ID:        "submission-1",
		CreatedAt: time.Now(),
		Resources: []schema.ResourceDef{},
	}

	valuesMap := map[string]any{}
	valuesMap["size"] = instance.InstanceType
	valuesMap["sku"] = instance.InstanceType
	valuesMap["location"] = instance.Region
	valuesMap["instances"] = 1

	valuesMap["pennywise_usage"] = map[string]any{}

	req.Resources = append(req.Resources, schema.ResourceDef{
		Address:      instance.Id,
		Type:         "azurerm_linux_virtual_machine_scale_set",
		Name:         "",
		RegionCode:   instance.Region,
		ProviderName: "azurerm",
		Values:       valuesMap,
	})

	reqBody, err := json.Marshal(req)
	if err != nil {
		return 0, err
	}

	var response cost.State
	statusCode, err := httpclient.DoRequest(ctx, "GET", s.pennywiseBaseUrl+"/api/v1/cost/submission", nil, reqBody, &response)
	if err != nil {
		return 0, err
	}

	if statusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to get pennywise cost, status code = %d", statusCode)
	}

	for _, comp := range response.GetCostComponents() {
		fmt.Println(comp.Name, comp.Unit, comp.Rate.InexactFloat64(), comp.Error, comp.MonthlyQuantity.InexactFloat64())
	}
	resourceCost, err := response.Cost()
	if err != nil {
		return 0, err
	}

	return resourceCost.Decimal.InexactFloat64(), nil
}
