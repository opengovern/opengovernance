package describer

import (
	"context"
	"strings"

	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/monitor/mgmt/insights"
	"github.com/Azure/azure-sdk-for-go/services/search/mgmt/2020-08-01/search"
	"github.com/Azure/go-autorest/autorest"
	"gitlab.com/keibiengine/keibi-engine/pkg/azure/model"
)

func SearchService(ctx context.Context, authorizer autorest.Authorizer, subscription string) ([]Resource, error) {
	client := insights.NewDiagnosticSettingsClient(subscription)
	client.Authorizer = authorizer

	searchClient := search.NewServicesClient(subscription)
	searchClient.Authorizer = authorizer

	result, err := searchClient.ListBySubscription(ctx, nil)
	if err != nil {
		return nil, err
	}
	var values []Resource
	for {
		for _, service := range result.Values() {
			resourceGroup := strings.Split(*service.ID, "/")[4]

			id := service.ID
			searchListOp, err := client.List(ctx, *id)
			if err != nil {
				return nil, err
			}
			values = append(values, Resource{
				ID:       *service.ID,
				Name:     *service.Name,
				Location: *service.Location,
				Description: model.SearchServiceDescription{
					Service:                     service,
					DiagnosticSettingsResources: searchListOp.Value,
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
