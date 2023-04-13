package describer

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/provisioningservices/mgmt/iothub"
	"github.com/Azure/azure-sdk-for-go/services/iothub/mgmt/2020-03-01/devices"
	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2021-04-01-preview/insights"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

func DevicesProvisioningServicesCertificates(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	rgs, err := listResourceGroups(ctx, authorizer, subscription)
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
					Name:        *v.Name,
					Location:    "global",
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

func IOTHub(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := insights.NewDiagnosticSettingsClient(subscription)
	client.Authorizer = authorizer

	iotHubClient := devices.NewIotHubResourceClient(subscription)
	iotHubClient.Authorizer = authorizer

	result, err := iotHubClient.ListBySubscription(ctx)
	if err != nil {
		return nil, err
	}
	var values []Resource
	for {
		for _, iotHubDescription := range result.Values() {
			resourceGroup := strings.Split(*iotHubDescription.ID, "/")[4]

			id := *iotHubDescription.ID

			devicesListOp, err := client.List(ctx, id)
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ID:       *iotHubDescription.ID,
				Name:     *iotHubDescription.Name,
				Location: *iotHubDescription.Location,
				Description: model.IOTHubDescription{
					IotHubDescription:           iotHubDescription,
					DiagnosticSettingsResources: devicesListOp.Value,
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

func IOTHubDps(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := insights.NewDiagnosticSettingsClient(subscription)
	client.Authorizer = authorizer

	iotHubClient := iothub.NewIotDpsResourceClient(subscription)
	iotHubClient.Authorizer = authorizer

	result, err := iotHubClient.ListBySubscription(ctx)
	if err != nil {
		return nil, err
	}
	var values []Resource
	for {
		for _, v := range result.Values() {
			resourceGroup := strings.Split(*v.ID, "/")[4]

			id := *v.ID

			devicesListOp, err := client.List(ctx, id)
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ID:       *v.ID,
				Name:     *v.Name,
				Location: *v.Location,
				Description: model.IOTHubDpsDescription{
					IotHubDps:                   v,
					DiagnosticSettingsResources: devicesListOp.Value,
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
