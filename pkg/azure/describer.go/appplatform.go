package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/monitor/mgmt/insights"
	"github.com/Azure/azure-sdk-for-go/profiles/latest/resources/mgmt/resources"
	"github.com/Azure/azure-sdk-for-go/services/appplatform/mgmt/2020-07-01/appplatform"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

func SpringCloudService(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
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
					appplatformListOp, err := insightsClient.List(ctx, id)
					if err != nil {
						return nil, err
					}
					values = append(values, Resource{
						ID: *service.ID,
						Description: model.SpringCloudServiceDescription{
							service,
							appplatformListOp,
						},
					})
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
