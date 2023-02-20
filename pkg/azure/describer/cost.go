package describer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/costmanagement/mgmt/2019-11-01/costmanagement"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/date"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
	"gitlab.com/keibiengine/keibi-engine/pkg/describe/enums"
)

const resourceTypeDimension = "resourceType"
const subscriptionDimension = "SubscriptionId"

func cost(ctx context.Context, authorizer autorest.Authorizer, subscription string, from time.Time, to time.Time, dimension string) ([]model.CostManagementQueryRow, error) {
	client := costmanagement.NewQueryClient(subscription)
	client.Authorizer = authorizer

	scope := fmt.Sprintf("subscriptions/%s", subscription)

	groupings := []costmanagement.QueryGrouping{
		{
			Type: costmanagement.QueryColumnTypeDimension,
			Name: &dimension,
		},
	}

	costAggregationString := "Cost"

	var costs, err = client.Usage(ctx, scope, costmanagement.QueryDefinition{
		Type:      costmanagement.ExportTypeAmortizedCost,
		Timeframe: costmanagement.TimeframeTypeCustom,
		TimePeriod: &costmanagement.QueryTimePeriod{
			From: &date.Time{Time: from},
			To:   &date.Time{Time: to},
		},
		Dataset: &costmanagement.QueryDataset{
			Granularity: costmanagement.GranularityTypeDaily,
			Grouping:    &groupings,
			Aggregation: map[string]*costmanagement.QueryAggregation{
				"Cost": {
					Name:     &costAggregationString,
					Function: costmanagement.FunctionTypeSum,
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	mapResult := make([]map[string]any, 0)
	for _, row := range *costs.Rows {
		rowMap := make(map[string]any)
		for i, column := range *costs.Columns {
			rowMap[*column.Name] = row[i]
		}
		mapResult = append(mapResult, rowMap)
	}
	jsonMapResult, err := json.Marshal(mapResult)
	if err != nil {
		return nil, err
	}

	result := make([]model.CostManagementQueryRow, 0, len(mapResult))
	err = json.Unmarshal(jsonMapResult, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func DailyCostByResourceType(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := costmanagement.NewQueryClient(subscription)
	client.Authorizer = authorizer

	triggerType := GetTriggerTypeFromContext(ctx)
	from := time.Now().AddDate(0, 0, -7)
	if triggerType == enums.DescribeTriggerTypeInitialDiscovery {
		from = time.Now().AddDate(0, -1, -7)
	}

	costResult, err := cost(ctx, authorizer, subscription, from, time.Now(), resourceTypeDimension)
	if err != nil {
		return nil, err
	}
	var values []Resource
	for _, row := range costResult {
		values = append(values, Resource{
			ID: fmt.Sprintf("resource-cost-%s/%s-%d", subscription, *row.ResourceType, row.UsageDate),
			Description: model.CostManagementCostByResourceTypeDescription{
				CostManagementCostByResourceType: row,
			},
		})
	}

	return values, nil
}

func DailyCostBySubscription(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := costmanagement.NewQueryClient(subscription)
	client.Authorizer = authorizer

	triggerType := GetTriggerTypeFromContext(ctx)
	from := time.Now().AddDate(0, 0, -7)
	if triggerType == enums.DescribeTriggerTypeInitialDiscovery {
		from = time.Now().AddDate(0, -1, -7)
	}

	costResult, err := cost(ctx, authorizer, subscription, from, time.Now(), subscriptionDimension)
	if err != nil {
		return nil, err
	}
	var values []Resource
	for _, row := range costResult {
		values = append(values, Resource{
			ID: fmt.Sprintf("resource-cost-%s/%d", subscription, row.UsageDate),
			Description: model.CostManagementCostBySubscriptionDescription{
				CostManagementCostBySubscription: row,
			},
		})
	}

	return values, nil
}
