package describer

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/healthcareapis/mgmt/healthcareapis"
	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2021-04-01-preview/insights"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

func HealthcareService(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
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
			resourceGroup := strings.Split(*v.ID, "/")[4]

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

			resource := Resource{
				ID:       *v.ID,
				Name:     *v.Name,
				Location: *v.Location,
				Description: model.HealthcareServiceDescription{
					ServicesDescription:         v,
					DiagnosticSettingsResources: opValue,
					PrivateEndpointConnections:  opServiceValue,
					ResourceGroup:               resourceGroup,
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
