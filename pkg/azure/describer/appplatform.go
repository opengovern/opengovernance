package describer

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
	"github.com/Azure/azure-sdk-for-go/services/appplatform/mgmt/2020-07-01/appplatform"
	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2021-04-01-preview/insights"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

func SpringCloudService(ctx context.Context, authorizer autorest.Authorizer, subscription string, stream *StreamSender) ([]Resource, error) {
	resourcesClient := resources.NewGroupsClient(subscription)
	resourcesClient.Authorizer = authorizer

	insightsClient := insights.NewDiagnosticSettingsClient(subscription)
	insightsClient.Authorizer = authorizer

	client := appplatform.NewServicesClient(subscription)
	client.Authorizer = authorizer

	result, err := resourcesClient.List(ctx, "", nil)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, resourceGroup := range result.Values() {
			if resourceGroup.Name == nil {
				continue
			}

			res, err := client.List(ctx, *resourceGroup.Name)
			if err != nil {
				return nil, err
			}
			for {
				for _, service := range res.Values() {
					id := *service.ID
					splitID := strings.Split(*service.ID, "/")

					resourceGroup := splitID[4]
					appplatformListOp, err := insightsClient.List(ctx, id)
					if err != nil {
						return nil, err
					}
					resource := Resource{
						ID:       *service.ID,
						Name:     *service.Name,
						Location: *service.Location,
						Description: model.SpringCloudServiceDescription{
							ServiceResource:            service,
							DiagnosticSettingsResource: appplatformListOp.Value,
							ResourceGroup:              resourceGroup,
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
				if !res.NotDone() {
					break
				}
				err = res.NextWithContext(ctx)
				if err != nil {
					return nil, err
				}
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
