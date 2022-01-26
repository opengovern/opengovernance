package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/monitor/mgmt/insights"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/servicebus/mgmt/servicebus"
	previewservicebus "github.com/Azure/azure-sdk-for-go/services/preview/servicebus/mgmt/2021-06-01-preview/servicebus"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
	"strings"
)

func ServiceBusQueue(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	rgs, err := resourceGroup(ctx, authorizer, subscription)
	if err != nil {
		return nil, err
	}

	client := servicebus.NewQueuesClient(subscription)
	client.Authorizer = authorizer

	var values []Resource
	for _, rg := range rgs {
		ns, err := serviceBusNamespace(ctx, authorizer, subscription, *rg.Name)
		if err != nil {
			return nil, err
		}

		for _, n := range ns {
			it, err := client.ListByNamespaceComplete(ctx, *rg.Name, *n.Name, nil, nil)
			if err != nil {
				return nil, err
			}

			for v := it.Value(); it.NotDone(); v = it.Value() {
				values = append(values, Resource{
					ID:          *v.ID,
					Description: JSONAllFieldsMarshaller{Value: v},
				})

				err := it.NextWithContext(ctx)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return values, nil
}

func ServiceBusTopic(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	rgs, err := resourceGroup(ctx, authorizer, subscription)
	if err != nil {
		return nil, err
	}

	client := servicebus.NewTopicsClient(subscription)
	client.Authorizer = authorizer

	var values []Resource
	for _, rg := range rgs {
		ns, err := serviceBusNamespace(ctx, authorizer, subscription, *rg.Name)
		if err != nil {
			return nil, err
		}

		for _, n := range ns {
			it, err := client.ListByNamespaceComplete(ctx, *rg.Name, *n.Name, nil, nil)
			if err != nil {
				return nil, err
			}

			for v := it.Value(); it.NotDone(); v = it.Value() {
				values = append(values, Resource{
					ID:          *v.ID,
					Description: JSONAllFieldsMarshaller{Value: v},
				})

				err := it.NextWithContext(ctx)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return values, nil
}

func serviceBusNamespace(ctx context.Context, authorizer autorest.Authorizer, subscription string, resourceGroup string) ([]servicebus.SBNamespace, error) {
	client := servicebus.NewNamespacesClient(subscription)
	client.Authorizer = authorizer

	it, err := client.ListByResourceGroupComplete(ctx, resourceGroup)
	if err != nil {
		return nil, err
	}

	var values []servicebus.SBNamespace
	for v := it.Value(); it.NotDone(); v = it.Value() {
		values = append(values, v)

		err := it.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}

	return values, nil
}
func ServicebusNamespace(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	servicebusClient := previewservicebus.NewPrivateEndpointConnectionsClient(subscription)
	servicebusClient.Authorizer = authorizer

	namespaceClient := previewservicebus.NewNamespacesClient(subscription)
	namespaceClient.Authorizer = authorizer

	insightsClient := insights.NewDiagnosticSettingsClient(subscription)
	insightsClient.Authorizer = authorizer

	client := previewservicebus.NewNamespacesClient(subscription)
	client.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}
	var values []Resource
	for {
		for _, namespace := range result.Values() {
			resourceGroup := strings.Split(*namespace.ID, "/")[4]

			insightsListOp, err := insightsClient.List(ctx, *namespace.ID)
			if err != nil {
				return nil, err
			}

			servicebusGetNetworkRuleSetOp, err := namespaceClient.GetNetworkRuleSet(ctx, resourceGroup, *namespace.Name)
			if err != nil {
				return nil, err
			}

			servicebusListOp, err := servicebusClient.List(ctx, resourceGroup, *namespace.Name)
			if err != nil {
				return nil, err
			}
			v := servicebusListOp.Values()
			for servicebusListOp.NotDone() {
				err := servicebusListOp.NextWithContext(ctx)
				if err != nil {
					return nil, err
				}

				v = append(v, servicebusListOp.Values()...)
			}
			values = append(values, Resource{
				ID: *namespace.ID,
				Description: model.ServicebusNamespaceDescription{
					SBNamespace:                 namespace,
					DiagnosticSettingsResources: insightsListOp.Value,
					NetworkRuleSet:              servicebusGetNetworkRuleSetOp,
					PrivateEndpointConnections:  v,
					ResourceGroup:               resourceGroup,
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
