package describer

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/profiles/preview/preview/eventgrid/mgmt/eventgrid"
	"github.com/Azure/go-autorest/autorest"
)

func EventGridDomainTopic(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]interface{}, error) {
	rgs, err := resourceGroup(ctx, authorizer, subscription)
	if err != nil {
		return nil, err
	}

	client := eventgrid.NewDomainTopicsClient(subscription)
	client.Authorizer = authorizer

	var values []interface{}
	for _, rg := range rgs {
		domains, err := eventGridDomain(ctx, authorizer, subscription, *rg.Name)
		if err != nil {
			return nil, err
		}

		for _, domain := range domains {
			it, err := client.ListByDomainComplete(ctx, *rg.Name, *domain.Name, "", nil)
			if err != nil {
				return nil, err
			}

			for v := it.Value(); it.NotDone(); v = it.Value() {
				values = append(values, JSONAllFieldsMarshaller{Value: v})

				err := it.NextWithContext(ctx)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return values, nil
}

func eventGridDomain(ctx context.Context, authorizer autorest.Authorizer, subscription string, resourceGroup string) ([]eventgrid.Domain, error) {
	client := eventgrid.NewDomainsClient(subscription)
	client.Authorizer = authorizer

	it, err := client.ListByResourceGroupComplete(ctx, resourceGroup, "", nil)
	if err != nil {
		return nil, err
	}

	var values []eventgrid.Domain
	for v := it.Value(); it.NotDone(); v = it.Value() {
		values = append(values, v)

		err := it.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}

	return values, nil
}
