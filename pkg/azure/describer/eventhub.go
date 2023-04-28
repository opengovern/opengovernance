package describer

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/preview/eventhub/mgmt/2018-01-01-preview/eventhub"
	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2021-04-01-preview/insights"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

func EventhubNamespace(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
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
			v := eventhubListOp.Values()
			for eventhubListOp.NotDone() {
				err := eventhubListOp.NextWithContext(ctx)
				if err != nil {
					return nil, err
				}

				v = append(v, eventhubListOp.Values()...)
			}

			resource := Resource{
				ID:       *namespace.ID,
				Name:     *namespace.Name,
				Location: *namespace.Location,
				Description: model.EventhubNamespaceDescription{
					EHNamespace:                 namespace,
					DiagnosticSettingsResources: insightsListOp.Value,
					NetworkRuleSet:              eventhubGetNetworkRuleSetOp,
					PrivateEndpointConnection:   v,
					ResourceGroup:               resourceGroupName,
				},
			}
			if stream != nil {
				if err := (*stream)(resource); err != nil {
					return nil, err
				}
			} else {
				values = append(values, resource)
			}
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
