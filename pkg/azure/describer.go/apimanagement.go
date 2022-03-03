package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/monitor/mgmt/insights"
	"github.com/Azure/azure-sdk-for-go/services/apimanagement/mgmt/2020-12-01/apimanagement"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
	"strings"
)

func APIManagement(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	apiManagementClient := apimanagement.NewServiceClient(subscription)
	apiManagementClient.Authorizer = authorizer

	insightsClient := insights.NewDiagnosticSettingsClient(subscription)
	insightsClient.Authorizer = authorizer

	result, err := apiManagementClient.List(ctx)
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, apiManagement := range result.Values() {
			resourceGroup := strings.Split(*apiManagement.ID, "/")[4]

			op, err := insightsClient.List(ctx, *apiManagement.ID)
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ID:       *apiManagement.ID,
				Location: *apiManagement.Location,
				Description: model.APIManagementDescription{
					APIManagement:               apiManagement,
					DiagnosticSettingsResources: *op.Value,
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
