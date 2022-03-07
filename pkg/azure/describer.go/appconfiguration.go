package describer

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/monitor/mgmt/insights"
	"github.com/Azure/azure-sdk-for-go/services/appconfiguration/mgmt/2020-06-01/appconfiguration"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
	"strings"
)

func AppConfiguration(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	configurationStoresClient := appconfiguration.NewConfigurationStoresClient(subscription)
	configurationStoresClient.Authorizer = authorizer

	insightsClient := insights.NewDiagnosticSettingsClient(subscription)
	insightsClient.Authorizer = authorizer

	result, err := configurationStoresClient.List(ctx, "")
	if err != nil {
		return nil, err
	}

	var values []Resource
	for {
		for _, config := range result.Values() {
			resourceGroup := strings.Split(*config.ID, "/")[4]

			op, err := insightsClient.List(ctx, *config.ID)
			if err != nil {
				return nil, err
			}

			values = append(values, Resource{
				ID:       *config.ID,
				Name:     *config.Name,
				Location: *config.Location,
				Description: model.AppConfigurationDescription{
					ConfigurationStore:          config,
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
