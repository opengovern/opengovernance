package describer

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/servicebus/mgmt/servicebus"
	"github.com/Azure/go-autorest/autorest"
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
