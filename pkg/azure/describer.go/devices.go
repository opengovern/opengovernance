package describer

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/provisioningservices/mgmt/iothub"
	"github.com/Azure/go-autorest/autorest"
)

func DevicesProvisioningServicesCertificates(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	rgs, err := resourceGroup(ctx, authorizer, subscription)
	if err != nil {
		return nil, err
	}

	client := iothub.NewDpsCertificateClient(subscription)
	client.Authorizer = authorizer

	var values []Resource
	for _, rg := range rgs {
		dpss, err := devicesProvisioningServices(ctx, authorizer, subscription, *rg.Name)
		if err != nil {
			return nil, err
		}

		for _, dps := range dpss {
			it, err := client.List(ctx, *rg.Name, *dps.Name)
			if err != nil {
				return nil, err
			}

			if it.Value == nil {
				continue
			}

			for _, v := range *it.Value {
				values = append(values, Resource{
					ID:          *v.ID,
					Description: JSONAllFieldsMarshaller{Value: v},
				})
			}
		}
	}

	return values, nil

}

func devicesProvisioningServices(ctx context.Context, authorizer autorest.Authorizer, subscription string, resourceGroup string) ([]iothub.ProvisioningServiceDescription, error) {
	client := iothub.NewIotDpsResourceClient(subscription)
	client.Authorizer = authorizer

	it, err := client.ListByResourceGroupComplete(ctx, resourceGroup)
	if err != nil {
		return nil, err
	}

	var values []iothub.ProvisioningServiceDescription
	for v := it.Value(); it.NotDone(); v = it.Value() {
		values = append(values, v)

		err := it.NextWithContext(ctx)
		if err != nil {
			return nil, err
		}
	}

	return values, nil
}
