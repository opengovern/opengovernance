package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/monitor/mgmt/insights"
	"github.com/Azure/azure-sdk-for-go/services/preview/eventhub/mgmt/2018-01-01-preview/eventhub"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
	"strings"
)

func EventhubNamespace(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	eventhubClient := eventhub.NewPrivateEndpointConnectionsClient(subscription)
	eventhubClient.Authorizer = authorizer

	insightsClient := insights.NewDiagnosticSettingsClient(subscription)
	insightsClient.Authorizer = authorizer

	client := eventhub.NewNamespacesClient(subscription)
	client.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, namespace := range result.Values() {
			resourceGroupName := strings.Split(string(*namespace.ID), "/")[4]

			insightsListOp, err := insightsClient.List(ctx, *namespace.ID)
			if err != nil {
				return nil, err
			}

			eventhubGetNetworkRuleSetOp, err := client.GetNetworkRuleSet(ctx, resourceGroupName, *namespace.Name)
			if err != nil {
				return nil, err
			}

			eventhubListOp, err := eventhubClient.List(ctx, resourceGroupName, *namespace.Name)
			if err != nil {
				return nil, err
			}
			values = append(values, Resource{
				ID: *namespace.ID,
				Description: model.EventhubNamespaceDescription{
					namespace,
					insightsListOp,
					eventhubGetNetworkRuleSetOp,
					eventhubListOp,
				},
			})
		}
		if !result.NotDone() {
			break
		}
		err = result.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}
	return values, nil
}
