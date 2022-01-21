package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/monitor/mgmt/insights"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/healthcareapis/mgmt/healthcareapis"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
	"strings"
)

func HealthcareService(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := healthcareapis.NewServicesClient(subscription)
	client.Authorizer = authorizer

	dignosticSettingClient := insights.NewDiagnosticSettingsClient(subscription)
	dignosticSettingClient.Authorizer = authorizer

	serviceClient := healthcareapis.NewPrivateEndpointConnectionsClient(subscription)
	serviceClient.Authorizer = authorizer

	result, err := client.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, v := range result.Values() {
			var opValue *[]insights.DiagnosticSettingsResource
			var opServiceValue *[]healthcareapis.PrivateEndpointConnection
			if v.ID != nil {
				resourceId := v.ID

				op, err := dignosticSettingClient.List(ctx, *resourceId)
				if err != nil {
					return nil, err
				}
				opValue = op.Value

				if v.Name != nil {
					resourceGroup := strings.Split(*v.ID, "/")[4]
					resourceName := v.Name

					// SDK does not support pagination yet
					opService, err := serviceClient.ListByService(ctx, resourceGroup, *resourceName)
					if err != nil {
						return nil, err
					}

					opServiceValue = opService.Value
				}
			}

			values = append(values, Resource{
				ID: *v.ID,
				Description: JSONAllFieldsMarshaller{
					model.HealthcareServiceDescription{
						ServicesDescription:         v,
						DiagnosticSettingsResources: opValue,
						PrivateEndpointConnections:  opServiceValue,
					},
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
